[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$fixtureRoot = Join-Path ([IO.Path]::GetTempPath()) ('noa-context-' + [Guid]::NewGuid().ToString('N'))
$contextRoot = Join-Path $fixtureRoot 'context'
$exportRoot = Join-Path $fixtureRoot 'export'
New-Item -ItemType Directory -Path $contextRoot -Force | Out-Null
try {
    Copy-Item -LiteralPath (Join-Path $repoRoot '.dockerignore') -Destination $contextRoot
    Copy-Item -LiteralPath (Join-Path $repoRoot 'Dockerfile.api.dockerignore') -Destination $contextRoot
    # 只构建无敏感内容的夹具，不读取或复制工作区环境文件。
    Set-Content -LiteralPath (Join-Path $contextRoot 'Dockerfile.api') -Value "FROM scratch`nCOPY . /" -Encoding utf8
    $excluded = @(
        'security-audit/sentinel.txt', 'backups/sentinel.sql', 'security-audit/test.env',
        '.env', '.env.local', 'internal/nested/.env', 'internal/nested/.env.production',
        'internal/nested/test.env', 'internal/nested/cert.key', 'internal/nested/cert.pem',
        'internal/nested/dump_backup.sql', 'internal/nested/data.noa-backup'
    )
    $included = @('go.mod', 'go.sum', 'cmd/main.go', 'internal/config/config.go',
        'model/media.go', 'deploy/mysql/migrations/001_initial.sql', 'etc/notes-of-ashen.yaml')
    foreach ($relative in @($excluded + $included)) {
        $path = Join-Path $contextRoot $relative
        New-Item -ItemType Directory -Path (Split-Path -Parent $path) -Force | Out-Null
        Set-Content -LiteralPath $path -Value 'non-sensitive build context marker' -Encoding utf8
    }
    & docker build --file (Join-Path $contextRoot 'Dockerfile.api') --output "type=local,dest=$exportRoot" $contextRoot
    if ($LASTEXITCODE -ne 0) { throw 'Docker 构建上下文检查失败。' }
    foreach ($relative in $excluded) {
        if (Test-Path -LiteralPath (Join-Path $exportRoot $relative)) { throw "敏感路径被纳入上下文：$relative" }
    }
    foreach ($relative in $included) {
        if (-not (Test-Path -LiteralPath (Join-Path $exportRoot $relative))) { throw "必需构建输入被排除：$relative" }
    }
    Write-Host "Docker API 上下文通过：$($excluded.Count) 项排除、$($included.Count) 项必需输入。"
} finally {
    $resolved = [IO.Path]::GetFullPath($fixtureRoot)
    $expectedParent = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd([IO.Path]::DirectorySeparatorChar)
    if ((Split-Path -Parent $resolved) -ne $expectedParent -or (Split-Path -Leaf $resolved) -notlike 'noa-context-*') {
        throw '拒绝清理非本轮构建夹具目录。'
    }
    Remove-Item -LiteralPath $resolved -Recurse -Force
}
