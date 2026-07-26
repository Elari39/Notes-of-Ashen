[CmdletBinding(DefaultParameterSetName = "Release")]
param(
    # 发布使用的镜像 tag。默认 v<日期时间>-<git 短 SHA>。
    [Parameter(ParameterSetName = "Release")]
    [string]$Tag = "",

    # 回滚到指定 tag（要求该 tag 的本地镜像存在）。
    [Parameter(ParameterSetName = "Rollback", Mandatory)]
    [string]$Rollback,

    # 仅构建镜像与更新 .env，不执行 docker compose up。
    [Parameter(ParameterSetName = "Release")]
    [switch]$SkipDeploy
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$envFile = Join-Path $repoRoot ".env"
$historyFile = Join-Path $repoRoot "deploy\release-history.local.jsonl"
$appImages = @("notes-of-ashen-web", "notes-of-ashen-api")

function Assert-Command {
    param([Parameter(Mandatory)][string]$Name)

    if ($null -eq (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "未找到必需命令：$Name"
    }
}

function Invoke-Native {
    param(
        [Parameter(Mandatory)][string]$Description,
        [Parameter(Mandatory)][string[]]$Arguments
    )

    & docker @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Description 执行失败（退出码 $LASTEXITCODE）。"
    }
}

function Set-EnvImageTag {
    # 定向替换 .env 中的 IMAGE_TAG 行，不读取和展示其他内容。
    param([Parameter(Mandatory)][string]$Value)

    if (-not (Test-Path $envFile)) {
        throw "未找到 .env（$envFile）。请先从 .env.example 创建。"
    }
    $lines = Get-Content $envFile
    if ($lines -match '^IMAGE_TAG=') {
        $lines = $lines -replace '^IMAGE_TAG=.*', "IMAGE_TAG=$Value"
    } else {
        $lines += "IMAGE_TAG=$Value"
    }
    Set-Content -Path $envFile -Value $lines -Encoding utf8
    Write-Host "[release] .env 已更新：IMAGE_TAG=$Value"
}

function Get-ImageDigests {
    param([Parameter(Mandatory)][string]$ImageTag)

    $digests = [ordered]@{}
    foreach ($image in $appImages) {
        $imageId = & docker image inspect --format '{{.Id}}' "${image}:${ImageTag}" 2>$null
        if ($LASTEXITCODE -ne 0) {
            throw "本地不存在镜像 ${image}:${ImageTag}。"
        }
        $digests[$image] = "$imageId".Trim()
    }
    return $digests
}

function Write-ReleaseRecord {
    param(
        [Parameter(Mandatory)][string]$Action,
        [Parameter(Mandatory)][string]$ImageTag,
        [Parameter(Mandatory)]$Digests
    )

    $historyDirectory = Split-Path $historyFile -Parent
    if (-not (Test-Path $historyDirectory)) {
        New-Item -ItemType Directory -Force $historyDirectory | Out-Null
    }
    $record = [ordered]@{
        time      = (Get-Date).ToString("o")
        action    = $Action
        imageTag  = $ImageTag
        gitCommit = "$(& git -C $repoRoot rev-parse HEAD)".Trim()
        images    = $Digests
    }
    Add-Content -Path $historyFile -Value ($record | ConvertTo-Json -Compress) -Encoding utf8
    Write-Host "[release] 记录已追加：$historyFile"
}

Assert-Command docker
Assert-Command git

Push-Location $repoRoot
try {
    if ($PSCmdlet.ParameterSetName -eq "Rollback") {
        # 回滚：只切换 IMAGE_TAG 并重启，不重新构建。
        $digests = Get-ImageDigests -ImageTag $Rollback
        Set-EnvImageTag -Value $Rollback
        Invoke-Native -Description "docker compose up -d" -Arguments @("compose", "up", "-d")
        Write-ReleaseRecord -Action "rollback" -ImageTag $Rollback -Digests $digests
        Write-Host "[release] 已回滚到 $Rollback。"
        return
    }

    $dirty = @(& git -C $repoRoot status --porcelain)
    if ($dirty.Count -gt 0) {
        Write-Warning "工作区存在未提交改动，发布记录的 gitCommit 无法完整代表镜像内容。"
    }

    if ([string]::IsNullOrWhiteSpace($Tag)) {
        $shortSha = "$(& git -C $repoRoot rev-parse --short HEAD)".Trim()
        $Tag = "v$(Get-Date -Format 'yyyyMMdd-HHmm')-$shortSha"
    }
    if ($Tag -eq "latest") {
        throw "发布 tag 不允许使用 latest；请使用不可变版本号。"
    }

    Write-Host "[release] 使用镜像 tag：$Tag"
    # docker-compose.yml 中 image 使用 ${IMAGE_TAG:-latest}，构建时通过环境变量注入。
    $env:IMAGE_TAG = $Tag
    try {
        Invoke-Native -Description "docker compose build" -Arguments @("compose", "build")
    } finally {
        Remove-Item Env:\IMAGE_TAG -ErrorAction SilentlyContinue
    }

    $digests = Get-ImageDigests -ImageTag $Tag
    Set-EnvImageTag -Value $Tag

    if (-not $SkipDeploy) {
        Invoke-Native -Description "docker compose up -d" -Arguments @("compose", "up", "-d")
    } else {
        Write-Host "[release] 已跳过部署（-SkipDeploy）；后续执行 docker compose up -d 生效。"
    }

    Write-ReleaseRecord -Action "release" -ImageTag $Tag -Digests $digests
    Write-Host "[release] 完成。回滚示例：scripts/release.ps1 -Rollback <旧tag>"
} finally {
    Pop-Location
}
