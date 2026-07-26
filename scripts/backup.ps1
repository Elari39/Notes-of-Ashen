[CmdletBinding()]
param(
    # 备份输出目录。
    [string]$OutputDir = "",

    # 备份保留天数，超期的备份目录会被清理。
    [ValidateRange(1, 3650)]
    [int]$RetentionDays = 14,

    # 跳过媒体数据卷归档，只导出数据库。
    [switch]$SkipMedia
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
if ([string]::IsNullOrWhiteSpace($OutputDir)) {
    $OutputDir = Join-Path $repoRoot "backups"
}

function Assert-Command {
    param([Parameter(Mandatory)][string]$Name)

    if ($null -eq (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "未找到必需命令：$Name"
    }
}

Assert-Command docker

Push-Location $repoRoot
try {
    # Compose 项目名用于解析数据卷真实名称（<project>_goblog_media_data）。
    $composeConfig = docker compose config --format json | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose config 执行失败，请确认在项目根目录且 .env 存在。"
    }
    $projectName = $composeConfig.name
    $mediaVolume = "${projectName}_goblog_media_data"

    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $backupDirectory = Join-Path $OutputDir $timestamp
    New-Item -ItemType Directory -Force $backupDirectory | Out-Null
    Write-Host "[backup] 输出目录：$backupDirectory"

    # 1. 数据库导出。凭证复用 mysql 容器内环境变量，不在宿主机展开；
    # 先在容器内落盘再 docker compose cp，避免 PowerShell 管道破坏 gzip 二进制流。
    $sqlGz = Join-Path $backupDirectory "mysql-notes_of_ashen.sql.gz"
    docker compose exec -T mysql sh -ec 'mysqldump --single-transaction --routines --triggers -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" | gzip -c > /tmp/noa-backup.sql.gz'
    if ($LASTEXITCODE -ne 0) {
        throw "mysqldump 执行失败（退出码 $LASTEXITCODE）。请确认 mysql 服务处于运行状态。"
    }
    docker compose cp mysql:/tmp/noa-backup.sql.gz $sqlGz
    if ($LASTEXITCODE -ne 0) {
        throw "复制数据库导出文件失败（退出码 $LASTEXITCODE）。"
    }
    docker compose exec -T mysql rm -f /tmp/noa-backup.sql.gz
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "清理容器内临时导出文件失败，可手动删除 mysql:/tmp/noa-backup.sql.gz。"
    }
    if ((Get-Item $sqlGz).Length -lt 512) {
        throw "数据库导出文件异常偏小（$((Get-Item $sqlGz).Length) 字节），备份中止。"
    }
    Write-Host "[backup] 数据库导出完成：$sqlGz"

    # 2. 媒体数据卷归档。
    if (-not $SkipMedia) {
        $mediaTarGz = Join-Path $backupDirectory "media.tar.gz"
        docker run --rm -v "${mediaVolume}:/data:ro" -v "${backupDirectory}:/backup" alpine:3 sh -ec 'tar czf /backup/media.tar.gz -C /data .'
        if ($LASTEXITCODE -ne 0) {
            throw "媒体卷归档失败（退出码 $LASTEXITCODE）。请确认数据卷 $mediaVolume 存在。"
        }
        Write-Host "[backup] 媒体归档完成：$mediaTarGz"
    } else {
        Write-Host "[backup] 已跳过媒体归档（-SkipMedia）。"
    }

    # 3. SHA-256 校验清单。用 LF+无 BOM 写入，保证 Linux 端 sha256sum -c 可直接校验。
    $checksumFile = Join-Path $backupDirectory "SHA256SUMS.txt"
    $checksumLines = Get-ChildItem $backupDirectory -File |
        Where-Object { $_.Name -ne "SHA256SUMS.txt" } |
        ForEach-Object { "$((Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant())  $($_.Name)" }
    [System.IO.File]::WriteAllText($checksumFile, (($checksumLines -join "`n") + "`n"), [System.Text.UTF8Encoding]::new($false))
    Write-Host "[backup] 校验清单：$checksumFile"

    # 4. 按保留天数清理过期备份目录（目录名即时间戳）。
    $cutoff = (Get-Date).AddDays(-$RetentionDays)
    $expired = Get-ChildItem $OutputDir -Directory -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match '^\d{8}-\d{6}$' -and $_.CreationTime -lt $cutoff }
    foreach ($directory in $expired) {
        Remove-Item -Recurse -Force $directory.FullName -Confirm:$false
        Write-Host "[backup] 已清理过期备份：$($directory.Name)"
    }

    # 5. 摘要。
    $totalBytes = (Get-ChildItem $backupDirectory -File | Measure-Object Length -Sum).Sum
    Write-Host ("[backup] 完成：{0}，共 {1:N1} MiB。请将该目录同步到异地/对象存储。" -f $timestamp, ($totalBytes / 1MB))
} finally {
    Pop-Location
}
