[CmdletBinding(DefaultParameterSetName = "Release")]
param(
    # 发布使用的镜像 tag。默认 v<日期时间>-<git 短 SHA>。
    [Parameter(ParameterSetName = "Release")]
    [string]$Tag = "",

    # 回退到指定 tag（要求该 tag 的本地镜像存在）。
    [Parameter(ParameterSetName = "Rollback", Mandatory)]
    [string]$Rollback,

    # 明确确认目标镜像可能落后于数据库 schema；仅用于已验证备份恢复路径。
    [Parameter(ParameterSetName = "Rollback")]
    [switch]$AllowIncompatibleSchema,

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

function Read-VersionOutput {
    param(
        [Parameter(Mandatory)][object[]]$Output,
        [Parameter(Mandatory)][string]$Description
    )

    $matches = @($Output | ForEach-Object { "$($_)".Trim() } | Where-Object { $_ -match '^\d+$' })
    if ($matches.Count -ne 1) {
        throw "$Description 未返回可识别的迁移版本。请确认镜像包含版本查询命令；如需紧急回退，请显式使用 -AllowIncompatibleSchema 并先验证备份恢复路径。"
    }
    return [uint64]$matches[0]
}

function Get-EmbeddedMigrationVersion {
    param([Parameter(Mandatory)][string]$ImageTag)

    $output = & docker run --rm --entrypoint /app/notes-of-ashen "notes-of-ashen-api:$ImageTag" -migration-version 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "目标镜像 notes-of-ashen-api:$ImageTag 无法读取内置迁移版本；已阻止代码回退。请先使用兼容镜像或显式 -AllowIncompatibleSchema。"
    }
    return Read-VersionOutput -Output @($output) -Description "目标镜像迁移版本"
}

function Get-DatabaseMigrationVersion {
    # 使用当前 Compose API 镜像通过 APP_DATABASE_DSN 查询实际数据库，兼容外部 MySQL。
    $output = & docker compose run --rm --no-deps api -f /app/etc/notes-of-ashen.yaml -schema-version 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "无法读取当前数据库迁移版本；已阻止代码回退。请确认当前 API 镜像支持 -schema-version，或显式使用 -AllowIncompatibleSchema 并先验证备份恢复路径。"
    }
    return Read-VersionOutput -Output @($output) -Description "数据库迁移版本"
}

function Assert-RollbackSchemaCompatibility {
    param(
        [Parameter(Mandatory)][string]$ImageTag,
        [Parameter(Mandatory)][switch]$Allow
    )

    if ($Allow) {
        Write-Warning "已使用 -AllowIncompatibleSchema：仅执行代码回退，不会回滚数据库 schema。必须确认可用备份恢复路径。"
        return
    }

    $targetVersion = Get-EmbeddedMigrationVersion -ImageTag $ImageTag
    $databaseVersion = Get-DatabaseMigrationVersion
    if ($databaseVersion -gt $targetVersion) {
        throw "拒绝代码回退：数据库迁移版本 $databaseVersion 高于目标镜像 $ImageTag 的版本 $targetVersion。请恢复兼容备份、选择包含该迁移的镜像，或在已验证恢复路径后显式使用 -AllowIncompatibleSchema。"
    }
    Write-Host "[release] schema 兼容性检查通过：database=$databaseVersion target=$targetVersion"
}

Assert-Command docker
Assert-Command git

Push-Location $repoRoot
try {
    if ($PSCmdlet.ParameterSetName -eq "Rollback") {
        # 代码回退：只切换 IMAGE_TAG 并重启，不重新构建，也不回滚数据库 schema。
        $digests = Get-ImageDigests -ImageTag $Rollback
        Assert-RollbackSchemaCompatibility -ImageTag $Rollback -Allow:$AllowIncompatibleSchema
        Set-EnvImageTag -Value $Rollback
        Invoke-Native -Description "docker compose up -d" -Arguments @("compose", "up", "-d")
        Write-ReleaseRecord -Action "code-rollback" -ImageTag $Rollback -Digests $digests
        Write-Host "[release] 已完成代码回退到 $Rollback（数据库 schema 未回退）。"
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
    Write-Host "[release] 完成。代码回退示例：scripts/release.ps1 -Rollback <旧tag>"
} finally {
    Pop-Location
}
