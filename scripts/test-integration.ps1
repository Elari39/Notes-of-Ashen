[CmdletBinding()]
param(
    [ValidateSet("core", "extended")]
    [string]$Suite = "core"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$baseComposeFiles = @(
    (Join-Path $repoRoot "docker-compose.yml"),
    (Join-Path $repoRoot "deploy\test\docker-compose.test.yml")
)
$runId = "$(Get-Date -Format 'yyyyMMddHHmmss')-$([Guid]::NewGuid().ToString('N').Substring(0, 8))"
$temporaryRoot = if ([string]::IsNullOrWhiteSpace($env:RUNNER_TEMP)) {
    [System.IO.Path]::GetTempPath()
} else {
    $env:RUNNER_TEMP
}
$artifactRoot = Join-Path $temporaryRoot "notes-of-ashen-integration-$runId"
$runSucceeded = $false

function Assert-Command {
    param([Parameter(Mandatory)][string]$Name)

    if ($null -eq (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "未找到必需命令：$Name"
    }
}

function New-TestSecret {
    # 只使用十六进制字符，避免 DSN 和 Compose .env 转义歧义。
    return ([Guid]::NewGuid().ToString("N") + [Guid]::NewGuid().ToString("N"))
}

function Get-ComposeArguments {
    param([Parameter(Mandatory)]$Runtime)

    $arguments = @(
        "compose",
        "--env-file", $Runtime.EnvFile,
        "--project-name", $Runtime.Project
    )
    foreach ($composeFile in $Runtime.ComposeFiles) {
        $arguments += @("--file", $composeFile)
    }
    return $arguments
}

function Invoke-Compose {
    param(
        [Parameter(Mandatory)]$Runtime,
        [Parameter(Mandatory)][string[]]$Arguments
    )

    $composeArguments = Get-ComposeArguments -Runtime $Runtime
    & docker @composeArguments @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose $($Arguments -join ' ') 执行失败（退出码 $LASTEXITCODE）。"
    }
}

function Get-ComposePort {
    param(
        [Parameter(Mandatory)]$Runtime,
        [Parameter(Mandatory)][string]$Service,
        [Parameter(Mandatory)][int]$ContainerPort
    )

    $composeArguments = Get-ComposeArguments -Runtime $Runtime
    $addresses = @(& docker @composeArguments port $Service $ContainerPort)
    if ($LASTEXITCODE -ne 0) {
        throw "无法获取 $Service`:$ContainerPort 的宿主机端口。"
    }

    $address = @($addresses | Where-Object { $_ -match '^127\.0\.0\.1:' } | Select-Object -First 1)
    if ($address.Count -eq 0) {
        $address = @($addresses | Select-Object -First 1)
    }
    if ($address.Count -eq 0) {
        throw "无法解析 $Service`:$ContainerPort 的端口映射：$($addresses -join ', ')"
    }

    $portMatch = [regex]::Match([string]$address[0], ':(\d+)$')
    if (-not $portMatch.Success) {
        throw "无法解析 $Service`:$ContainerPort 的端口映射：$($addresses -join ', ')"
    }

    return [int]$portMatch.Groups[1].Value
}

function Get-ComposeContainerIds {
    param(
        [Parameter(Mandatory)]$Runtime,
        [Parameter(Mandatory)][string]$Service,
        [switch]$IncludeStopped
    )

    $composeArguments = Get-ComposeArguments -Runtime $Runtime
    $psArguments = @("ps")
    if ($IncludeStopped) {
        $psArguments += "--all"
    }
    $psArguments += @("--quiet", $Service)
    $containerLines = @(& docker @composeArguments @psArguments)
    $containerIds = @($containerLines | ForEach-Object { ([string]$_).Trim() } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    if ($LASTEXITCODE -ne 0 -or $containerIds.Count -eq 0) {
        throw "无法获取 $Service 容器 ID。"
    }

    return $containerIds
}

function Get-ComposeContainerId {
    param(
        [Parameter(Mandatory)]$Runtime,
        [Parameter(Mandatory)][string]$Service
    )

    $containerIds = @(Get-ComposeContainerIds -Runtime $Runtime -Service $Service)
    return $containerIds[0]
}

function New-StageRuntime {
    param(
        [Parameter(Mandatory)][string]$Stage,
        [Parameter(Mandatory)][int]$Ordinal,
        [string[]]$ComposeOverrides = @(),
        [System.Collections.IDictionary]$EnvironmentOverrides = @{}
    )

    $project = "noa-it-$($runId.Replace('-', ''))-$Stage-$Ordinal".ToLowerInvariant()
    # 测试网络与开发环境 172.30.127.0/24 分离；每个阶段使用新网段，降低并发运行冲突概率。
    $networkPart = Get-Random -Minimum 20 -Maximum 250
    $subnet = "172.28.$networkPart.0/24"
    $webIPv4 = "172.28.$networkPart.10"
    $gatewayIPv4 = "172.28.$networkPart.1"
    $mysqlPassword = New-TestSecret
    $mysqlRootPassword = New-TestSecret
    $envFile = Join-Path $artifactRoot "$Stage-$Ordinal.env"
    $imageTag = "integration-$($runId.Replace('-', ''))-$Stage-$Ordinal"

    $composeEnvironment = [ordered]@{
        APP_ENV_FILE                 = $envFile.Replace("\", "/")
        IMAGE_TAG                    = $imageTag
        WEB_PORT                     = "0"
        E2E_API_PORT                 = "0"
        E2E_MYSQL_PORT               = "0"
        E2E_REDIS_PORT               = "0"
        APP_HOST                     = "0.0.0.0"
        APP_PORT                     = "19000"
        APP_TIMEOUT                  = "610000"
        APP_DOCKER_SUBNET            = $subnet
        APP_WEB_IPV4_ADDRESS         = $webIPv4
        WEB_TRUSTED_PROXY_CIDR       = "$gatewayIPv4/32"
        APP_TRUSTED_PROXY_CIDRS      = "$webIPv4/32"
        APP_AUTH_ACCESS_SECRET       = (New-TestSecret)
        APP_AUTH_ACCESS_EXPIRE       = "1800"
        APP_AUTH_REFRESH_EXPIRE      = "604800"
        APP_AUTH_COOKIE_SECURE       = "false"
        APP_MYSQL_ROOT_PASSWORD      = $mysqlRootPassword
        APP_MYSQL_USER               = "notes_test"
        APP_MYSQL_PASSWORD           = $mysqlPassword
        APP_DATABASE_DSN             = "notes_test:$mysqlPassword@tcp(mysql:3306)/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local"
        APP_DATABASE_MAX_OPEN_CONNS  = "20"
        APP_DATABASE_MAX_IDLE_CONNS  = "10"
        APP_REDIS_ADDR               = "redis:6379"
        APP_REDIS_PASSWORD           = ""
        APP_REDIS_DB                 = "0"
        # 虽然测试 profile 不启动 RabbitMQ，Compose 仍会在解析服务时校验必填插值。
        APP_RABBITMQ_USER            = "notes_test"
        APP_RABBITMQ_PASSWORD        = ""
        APP_RABBITMQ_ENABLED         = "false"
        APP_RABBITMQ_URL             = ""
        APP_SEARCH_ENABLED           = "false"
        APP_MEILISEARCH_HOST         = "http://meilisearch:7700"
        APP_MEILISEARCH_API_KEY      = ""
        APP_MEILISEARCH_INDEX        = "articles"
        APP_EMAIL_ENABLED            = "false"
        APP_EMAIL_SMTP_HOST          = ""
        APP_EMAIL_SMTP_PORT          = "465"
        APP_EMAIL_TLS_MODE           = "implicit"
        APP_EMAIL_SMTP_USERNAME      = ""
        APP_EMAIL_SMTP_PASSWORD      = ""
        APP_EMAIL_FROM               = ""
        APP_EMAIL_FROM_NAME          = "Notes of Ashen Integration"
        APP_MEDIA_ROOT               = "/data/media"
        APP_MEDIA_MAX_UPLOAD_BYTES   = "10485760"
        APP_BACKUP_MAX_UPLOAD_BYTES  = "1073741824"
        WEB_BACKUP_MAX_BODY_SIZE     = "1025m"
        PRERENDER_ENABLED            = "0"
        PRERENDER_SERVICE_URL        = "http://prerender.invalid"
        PRERENDER_TOKEN              = ""
    }

    foreach ($entry in $EnvironmentOverrides.GetEnumerator()) {
        $composeEnvironment[[string]$entry.Key] = [string]$entry.Value
    }

    $envLines = foreach ($entry in $composeEnvironment.GetEnumerator()) {
        "$($entry.Key)=$($entry.Value)"
    }
    Set-Content -LiteralPath $envFile -Value $envLines -Encoding utf8

    return [PSCustomObject]@{
        Stage              = $Stage
        Project            = $project
        EnvFile            = $envFile
        ArtifactDirectory  = (Join-Path $artifactRoot $Stage)
        ComposeEnvironment = $composeEnvironment
        ComposeFiles        = @($baseComposeFiles + $ComposeOverrides)
        MySQLRootPassword  = $mysqlRootPassword
        RedisPassword      = [string]$composeEnvironment["APP_REDIS_PASSWORD"]
    }
}

function Set-EnvironmentValue {
    param(
        [Parameter(Mandatory)][hashtable]$Snapshot,
        [Parameter(Mandatory)][string]$Name,
        [AllowEmptyString()][string]$Value
    )

    if (-not $Snapshot.ContainsKey($Name)) {
        $exists = Test-Path -LiteralPath "Env:$Name"
        $Snapshot[$Name] = [PSCustomObject]@{
            Exists = $exists
            Value  = if ($exists) { (Get-Item -LiteralPath "Env:$Name").Value } else { $null }
        }
    }
    Set-Item -LiteralPath "Env:$Name" -Value $Value
}

function Restore-Environment {
    param([Parameter(Mandatory)][hashtable]$Snapshot)

    foreach ($entry in $Snapshot.GetEnumerator()) {
        if ($entry.Value.Exists) {
            Set-Item -LiteralPath "Env:$($entry.Key)" -Value $entry.Value.Value
        } else {
            Remove-Item -LiteralPath "Env:$($entry.Key)" -ErrorAction SilentlyContinue
        }
    }
}

function Prepare-StageEnvironment {
    param(
        [Parameter(Mandatory)]$Runtime,
        [Parameter(Mandatory)][hashtable]$Snapshot
    )

    foreach ($entry in $Runtime.ComposeEnvironment.GetEnumerator()) {
        Set-EnvironmentValue -Snapshot $Snapshot -Name $entry.Key -Value ([string]$entry.Value)
    }

    # 显式 -f / --env-file 已提供所有输入；屏蔽调用者的 Compose 环境，避免意外加载开发配置。
    Set-EnvironmentValue -Snapshot $Snapshot -Name "COMPOSE_FILE" -Value ""
    Set-EnvironmentValue -Snapshot $Snapshot -Name "COMPOSE_PROJECT_NAME" -Value $Runtime.Project
    Set-EnvironmentValue -Snapshot $Snapshot -Name "COMPOSE_PROFILES" -Value ""
    Set-EnvironmentValue -Snapshot $Snapshot -Name "COMPOSE_ENV_FILES" -Value ""
    Set-EnvironmentValue -Snapshot $Snapshot -Name "COMPOSE_DISABLE_ENV_FILE" -Value "1"
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_ARTIFACT_DIR" -Value $Runtime.ArtifactDirectory
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_COMPOSE_PROJECT" -Value $Runtime.Project
    New-Item -ItemType Directory -Force -Path $Runtime.ArtifactDirectory | Out-Null
}

function Wait-ForApi {
    param([Parameter(Mandatory)]$Runtime)

    $deadline = (Get-Date).AddSeconds(180)
    while ((Get-Date) -lt $deadline) {
        try {
            $response = Invoke-WebRequest -Uri "$($Runtime.ApiBaseUrl)/healthz" -Method Get -TimeoutSec 5
            if ($response.StatusCode -eq 200) {
                return
            }
        } catch {
            # Docker healthcheck 和 schema 初始化尚未完成时继续等待。
        }
        Start-Sleep -Seconds 2
    }

    throw "等待 API 健康检查超时：$($Runtime.ApiBaseUrl)/healthz"
}

function Wait-ForWeb {
    param([Parameter(Mandatory)]$Runtime)

    $deadline = (Get-Date).AddSeconds(180)
    while ((Get-Date) -lt $deadline) {
        try {
            $response = Invoke-WebRequest -Uri "$($Runtime.WebBaseUrl)/" -Method Get -TimeoutSec 5
            if ($response.StatusCode -eq 200) {
                return
            }
        } catch {
            # Nginx 静态资源和其依赖的 API 健康检查尚未完成时继续等待。
        }
        Start-Sleep -Seconds 2
    }

    throw "等待 Web 健康检查超时：$($Runtime.WebBaseUrl)/"
}

function Wait-ForContainerHealthy {
    param(
        [Parameter(Mandatory)][string]$ContainerID,
        [Parameter(Mandatory)][string]$Service
    )

    $deadline = (Get-Date).AddSeconds(180)
    $lastStatus = "unknown"
    while ((Get-Date) -lt $deadline) {
        $status = @(& docker inspect --format "{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}" $ContainerID)
        if ($LASTEXITCODE -eq 0 -and $status.Count -gt 0) {
            $lastStatus = ([string]$status[0]).Trim()
            if ($lastStatus -eq "healthy") {
                return
            }
        }
        Start-Sleep -Seconds 2
    }

    throw "等待 $Service 容器健康检查超时（最后状态：$lastStatus）。"
}

function Wait-ForContainerExit {
    param(
        [Parameter(Mandatory)][string]$ContainerID,
        [Parameter(Mandatory)][string]$Service
    )

    $deadline = (Get-Date).AddSeconds(90)
    $lastStatus = "unknown"
    while ((Get-Date) -lt $deadline) {
        $state = @(& docker inspect --format "{{.State.Status}}|{{.State.ExitCode}}" $ContainerID)
        if ($LASTEXITCODE -eq 0 -and $state.Count -gt 0) {
            $parts = ([string]$state[0]).Trim().Split("|", 2)
            $lastStatus = $parts[0]
            if ($lastStatus -eq "exited" -or $lastStatus -eq "dead") {
                $exitCode = if ($parts.Count -eq 2) { [int]$parts[1] } else { -1 }
                return [PSCustomObject]@{
                    Status   = $lastStatus
                    ExitCode = $exitCode
                }
            }
        }
        Start-Sleep -Seconds 2
    }

    throw "等待 $Service 容器退出超时（最后状态：$lastStatus）。"
}

function Wait-ForComposeContainerIds {
    param(
        [Parameter(Mandatory)]$Runtime,
        [Parameter(Mandatory)][string]$Service,
        [switch]$IncludeStopped
    )

    $deadline = (Get-Date).AddSeconds(90)
    while ((Get-Date) -lt $deadline) {
        try {
            $containerIds = @(Get-ComposeContainerIds -Runtime $Runtime -Service $Service -IncludeStopped:$IncludeStopped)
            if ($containerIds.Count -gt 0) {
                return $containerIds
            }
        } catch {
            # 依赖服务刚被 Compose 创建时继续等待。
        }
        Start-Sleep -Seconds 2
    }

    throw "等待 $Service 容器创建超时。"
}

function Initialize-DatabaseEndpoints {
    param([Parameter(Mandatory)]$Runtime)

    $Runtime | Add-Member -Force -NotePropertyName MySQLPort -NotePropertyValue (Get-ComposePort -Runtime $Runtime -Service "mysql" -ContainerPort 3306)
    $Runtime | Add-Member -Force -NotePropertyName RedisPort -NotePropertyValue (Get-ComposePort -Runtime $Runtime -Service "redis" -ContainerPort 6379)
    $Runtime | Add-Member -Force -NotePropertyName RedisContainerId -NotePropertyValue (Get-ComposeContainerId -Runtime $Runtime -Service "redis")
    $Runtime | Add-Member -Force -NotePropertyName MySQLContainerId -NotePropertyValue (Get-ComposeContainerId -Runtime $Runtime -Service "mysql")
}

function Initialize-HttpEndpoints {
    param([Parameter(Mandatory)]$Runtime)

    $Runtime | Add-Member -Force -NotePropertyName WebPort -NotePropertyValue (Get-ComposePort -Runtime $Runtime -Service "web" -ContainerPort 8080)
    $Runtime | Add-Member -Force -NotePropertyName ApiPort -NotePropertyValue (Get-ComposePort -Runtime $Runtime -Service "api" -ContainerPort 19000)
    $Runtime | Add-Member -Force -NotePropertyName WebBaseUrl -NotePropertyValue "http://127.0.0.1:$($Runtime.WebPort)"
    $Runtime | Add-Member -Force -NotePropertyName ApiBaseUrl -NotePropertyValue "http://127.0.0.1:$($Runtime.ApiPort)"
}

function Start-Stage {
    param([Parameter(Mandatory)]$Runtime)

    Invoke-Compose -Runtime $Runtime -Arguments @("up", "--detach", "--build", "web", "api", "mysql", "redis")
    Initialize-DatabaseEndpoints -Runtime $Runtime
    Initialize-HttpEndpoints -Runtime $Runtime

    Wait-ForApi -Runtime $Runtime
    Wait-ForWeb -Runtime $Runtime
}

function Start-MigrationPrerequisites {
    param([Parameter(Mandatory)]$Runtime)

    # 只启动 MySQL/Redis，避免 api 的 depends_on 自动拉起 migrate；这样才能让两个
    # 明确启动的 migrate job 在旧库上竞争同一个 MySQL advisory lock。
    Invoke-Compose -Runtime $Runtime -Arguments @("up", "--detach", "mysql", "redis")
    Initialize-DatabaseEndpoints -Runtime $Runtime
    Wait-ForContainerHealthy -ContainerID $Runtime.MySQLContainerId -Service "mysql"
    Wait-ForContainerHealthy -ContainerID $Runtime.RedisContainerId -Service "redis"
    Invoke-Compose -Runtime $Runtime -Arguments @("build", "migrate")
}

function Start-ApiAndWebAfterMigration {
    param([Parameter(Mandatory)]$Runtime)

    # 此处仍保留 Compose 的 migrate 依赖链；已由并发 job 完成的版本不得再产生执行记录。
    # 先缩放 API 到两个副本，确保两个实例都只会在迁移完成后进入健康状态。
    Invoke-Compose -Runtime $Runtime -Arguments @("up", "--detach", "--scale", "api=2", "api")
    $apiContainerIds = @(Get-ComposeContainerIds -Runtime $Runtime -Service "api")
    if ($apiContainerIds.Count -ne 2) {
        throw "迁移完成后 API 副本数 = $($apiContainerIds.Count)，期望 2。"
    }
    foreach ($apiContainerId in $apiContainerIds) {
        Wait-ForContainerHealthy -ContainerID $apiContainerId -Service "api"
    }

    # 不重新解析 web 的 api 依赖，避免 Compose 按默认 scale=1 缩容已验证健康的两个副本。
    Invoke-Compose -Runtime $Runtime -Arguments @("up", "--detach", "--no-deps", "web")
    Initialize-HttpEndpoints -Runtime $Runtime
    Wait-ForApi -Runtime $Runtime
    Wait-ForWeb -Runtime $Runtime
}

function Set-TestEndpoints {
    param(
        [Parameter(Mandatory)]$Runtime,
        [Parameter(Mandatory)][hashtable]$Snapshot
    )

    $redisPassword = [string]$Runtime.RedisPassword
    $redisURL = if ([string]::IsNullOrEmpty($redisPassword)) {
        "redis://127.0.0.1:$($Runtime.RedisPort)/0"
    } else {
        "redis://:$([uri]::EscapeDataString($redisPassword))@127.0.0.1:$($Runtime.RedisPort)/0"
    }

    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_WEB_BASE_URL" -Value $Runtime.WebBaseUrl
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_API_BASE_URL" -Value $Runtime.ApiBaseUrl
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_REDIS_URL" -Value $redisURL
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_REDIS_PASSWORD" -Value $redisPassword
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_MYSQL_DSN" -Value "notes_test:$($Runtime.ComposeEnvironment['APP_MYSQL_PASSWORD'])@tcp(127.0.0.1:$($Runtime.MySQLPort))/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local"
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_MYSQL_ROOT_DSN" -Value "root:$($Runtime.MySQLRootPassword)@tcp(127.0.0.1:$($Runtime.MySQLPort))/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local"
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_REDIS_CONTAINER_ID" -Value $Runtime.RedisContainerId
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_MYSQL_CONTAINER_ID" -Value $Runtime.MySQLContainerId
}

function Invoke-RedisCli {
    param(
        [Parameter(Mandatory)]$Runtime,
        [bool]$UsePassword = $false
    )

    $arguments = @("exec")
    $previousPassword = $null
    try {
        if ($UsePassword) {
            $previousPassword = [Environment]::GetEnvironmentVariable("REDISCLI_AUTH", "Process")
            [Environment]::SetEnvironmentVariable("REDISCLI_AUTH", [string]$Runtime.RedisPassword, "Process")
            # 让 Docker CLI 从当前进程环境转发变量，避免密码出现在命令参数或错误输出中。
            $arguments += @("-e", "REDISCLI_AUTH")
        }
        $arguments += @($Runtime.RedisContainerId, "redis-cli", "ping")
        $output = @(& docker @arguments 2>&1)
        return [PSCustomObject]@{
            ExitCode = $LASTEXITCODE
            Output   = (($output | ForEach-Object { $_.ToString() }) -join "`n").Trim()
        }
    } finally {
        if ($UsePassword) {
            [Environment]::SetEnvironmentVariable("REDISCLI_AUTH", $previousPassword, "Process")
        }
    }
}

function Assert-RedisPasswordMode {
    param([Parameter(Mandatory)]$Runtime)

    $password = [string]$Runtime.RedisPassword
    if ([string]::IsNullOrEmpty($password)) {
        $anonymous = Invoke-RedisCli -Runtime $Runtime
        if ($anonymous.ExitCode -ne 0 -or $anonymous.Output -notmatch "(?m)^PONG$") {
            throw "空 APP_REDIS_PASSWORD 时 Redis 未允许无认证 PING：exit=$($anonymous.ExitCode) output=$($anonymous.Output)"
        }
        return
    }

    $anonymous = Invoke-RedisCli -Runtime $Runtime
    if ($anonymous.Output -notmatch "(?i)(NOAUTH|AUTH.*required|authentication required)") {
        throw "非空 APP_REDIS_PASSWORD 时 Redis 未拒绝无认证 PING：exit=$($anonymous.ExitCode) output=$($anonymous.Output)"
    }

    $authenticated = Invoke-RedisCli -Runtime $Runtime -UsePassword $true
    if ($authenticated.ExitCode -ne 0 -or $authenticated.Output -notmatch "(?m)^PONG$") {
        throw "非空 APP_REDIS_PASSWORD 时 Redis 未接受正确认证：exit=$($authenticated.ExitCode) output=$($authenticated.Output)"
    }
}

function Get-ComposeServiceLogs {
    param(
        [Parameter(Mandatory)]$Runtime,
        [Parameter(Mandatory)][string]$Service
    )

    $composeArguments = Get-ComposeArguments -Runtime $Runtime
    $output = @(& docker @composeArguments logs --no-color $Service 2>&1)
    if ($LASTEXITCODE -ne 0) {
        throw "读取 $Service Compose 日志失败。"
    }
    return (($output | ForEach-Object { $_.ToString() }) -join "`n")
}

function Get-ExpectedMigrations {
    $migrationDirectory = Join-Path $repoRoot "deploy\mysql\migrations"
    $items = @(
        Get-ChildItem -LiteralPath $migrationDirectory -Filter "*.sql" -File |
            ForEach-Object {
                $match = [regex]::Match($_.Name, '^(\d{6})_[A-Za-z0-9][A-Za-z0-9_-]*\.sql$')
                if (-not $match.Success) {
                    throw "发现无效迁移文件名：$($_.Name)"
                }
                [PSCustomObject]@{
                    Version  = [int]$match.Groups[1].Value
                    Name     = $_.Name
                    Checksum = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
                }
            } |
            Sort-Object Version
    )
    if ($items.Count -eq 0) {
        throw "未找到正式迁移文件：$migrationDirectory"
    }
    for ($index = 0; $index -lt $items.Count; $index++) {
        $expectedVersion = $index + 1
        if ($items[$index].Version -ne $expectedVersion) {
            throw "迁移文件版本不连续：期望 $expectedVersion，实际 $($items[$index].Version)"
        }
    }
    return $items
}

function Invoke-MySQLQuery {
    param(
        [Parameter(Mandatory)]$Runtime,
        [Parameter(Mandatory)][string]$Query
    )

    # 密码只在隔离容器进程环境中传递，避免出现在命令参数和失败日志里。
    $output = @(& docker exec -e "MYSQL_PWD=$($Runtime.MySQLRootPassword)" $Runtime.MySQLContainerId mysql --protocol=TCP -h 127.0.0.1 -u root -N -B notes_of_ashen -e $Query)
    if ($LASTEXITCODE -ne 0) {
        throw "执行迁移验证 MySQL 查询失败。"
    }
    return @($output | ForEach-Object { ([string]$_).Trim() } | Where-Object { $_ -ne "" })
}

function Get-MigrationRunSnapshot {
    param([Parameter(Mandatory)]$Runtime)

    return @(Invoke-MySQLQuery -Runtime $Runtime -Query "SELECT version, status, COUNT(*) FROM schema_migration_runs GROUP BY version, status ORDER BY version, status")
}

function Assert-MigrationState {
    param([Parameter(Mandatory)]$Runtime)

    $expected = @(Get-ExpectedMigrations)
    $metadata = @(Invoke-MySQLQuery -Runtime $Runtime -Query "SELECT version, name, checksum FROM schema_migrations ORDER BY version")
    if ($metadata.Count -ne $expected.Count) {
        throw "schema_migrations 记录数 = $($metadata.Count)，期望 $($expected.Count)。"
    }
    for ($index = 0; $index -lt $expected.Count; $index++) {
        $parts = $metadata[$index] -split "`t", 3
        if ($parts.Count -ne 3) {
            throw "schema_migrations 返回格式无效：$($metadata[$index])"
        }
        $item = $expected[$index]
        if ([int]$parts[0] -ne $item.Version -or $parts[1] -ne $item.Name -or $parts[2].ToLowerInvariant() -ne $item.Checksum) {
            throw "schema_migrations 第 $($index + 1) 条与内置迁移不一致：实际 '$($metadata[$index])'，期望 '$($item.Version)`t$($item.Name)`t$($item.Checksum)'。"
        }
    }

    $runs = @(Get-MigrationRunSnapshot -Runtime $Runtime)
    if ($runs.Count -ne $expected.Count) {
        throw "schema_migration_runs 分组数 = $($runs.Count)，期望每个版本恰有一条成功记录（$($expected.Count) 条）。"
    }
    for ($index = 0; $index -lt $expected.Count; $index++) {
        $parts = $runs[$index] -split "`t", 3
        if ($parts.Count -ne 3 -or [int]$parts[0] -ne $expected[$index].Version -or $parts[1] -ne "success" -or [int]$parts[2] -ne 1) {
            throw "schema_migration_runs 第 $($index + 1) 条不满足每个版本仅执行一次：$($runs[$index])"
        }
    }

    $failedCount = @(Invoke-MySQLQuery -Runtime $Runtime -Query "SELECT COUNT(*) FROM schema_migration_runs WHERE status <> 'success'")
    if ($failedCount.Count -ne 1 -or [int]$failedCount[0] -ne 0) {
        throw "迁移执行记录包含失败状态：$($failedCount -join ', ')"
    }
}

function Assert-ApiMigrationHealth {
    param([Parameter(Mandatory)]$Runtime)

    try {
        $response = Invoke-WebRequest -Uri "$($Runtime.ApiBaseUrl)/healthz" -Method Get -TimeoutSec 10
    } catch {
        throw "迁移完成后 API /healthz 未返回 200：$($_.Exception.Message)"
    }
    if ($response.StatusCode -ne 200) {
        throw "迁移完成后 API /healthz 状态码 = $($response.StatusCode)，期望 200。"
    }
    try {
        $report = $response.Content | ConvertFrom-Json
    } catch {
        throw "迁移完成后 API /healthz JSON 无法解析：$($_.Exception.Message)"
    }
    if ($report.status -ne "ok" -or $null -eq $report.checks -or $report.checks.schema.status -ne "up") {
        throw "迁移完成后 API schema 健康检查异常：$($response.Content)"
    }
}

function Invoke-MigrateTask {
    param([Parameter(Mandatory)]$Runtime)

    Invoke-Compose -Runtime $Runtime -Arguments @("run", "--rm", "--no-deps", "migrate")
}

function Start-ComposeProcess {
    param(
        [Parameter(Mandatory)]$Runtime,
        [Parameter(Mandatory)][string[]]$Arguments
    )

    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = "docker"
    $startInfo.UseShellExecute = $false
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    foreach ($argument in @((Get-ComposeArguments -Runtime $Runtime) + $Arguments)) {
        [void]$startInfo.ArgumentList.Add([string]$argument)
    }
    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    if (-not $process.Start()) {
        throw "无法启动并发 docker compose migrate 进程。"
    }
    return [PSCustomObject]@{
        Process = $process
        Output  = $process.StandardOutput.ReadToEndAsync()
        Error   = $process.StandardError.ReadToEndAsync()
    }
}

function Wait-ComposeProcess {
    param([Parameter(Mandatory)]$ProcessInfo)

    $ProcessInfo.Process.WaitForExit()
    $output = $ProcessInfo.Output.GetAwaiter().GetResult()
    $errorOutput = $ProcessInfo.Error.GetAwaiter().GetResult()
    if ($ProcessInfo.Process.ExitCode -ne 0) {
        throw "并发 migrate job 退出码 $($ProcessInfo.Process.ExitCode)：$($output.Trim()) $($errorOutput.Trim())"
    }
}

function Invoke-ConcurrentMigrateTasks {
    param([Parameter(Mandatory)]$Runtime)

    $arguments = @("run", "--rm", "--no-deps", "migrate")
    $first = Start-ComposeProcess -Runtime $Runtime -Arguments $arguments
    $second = Start-ComposeProcess -Runtime $Runtime -Arguments $arguments
    Wait-ComposeProcess -ProcessInfo $first
    Wait-ComposeProcess -ProcessInfo $second
}

function Save-StageLogs {
    param([Parameter(Mandatory)]$Runtime)

    try {
        $composeArguments = Get-ComposeArguments -Runtime $Runtime
        & docker @composeArguments ps --all *> (Join-Path $Runtime.ArtifactDirectory "compose-ps.log")
        & docker @composeArguments logs --no-color --timestamps *> (Join-Path $Runtime.ArtifactDirectory "compose.log")
    } catch {
        Write-Warning "保存 $($Runtime.Stage) 阶段 Compose 日志失败：$($_.Exception.Message)"
    }
}

function Stop-Stage {
    param(
        [Parameter(Mandatory)]$Runtime,
        [bool]$KeepLogs
    )

    if ($KeepLogs) {
        Save-StageLogs -Runtime $Runtime
    }

    try {
        $composeArguments = Get-ComposeArguments -Runtime $Runtime
        & docker @composeArguments down --volumes --remove-orphans --rmi local
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "清理测试 Compose 项目 $($Runtime.Project) 失败（退出码 $LASTEXITCODE）。"
        }
    } catch {
        Write-Warning "清理测试 Compose 项目 $($Runtime.Project) 失败：$($_.Exception.Message)"
    }
}

function Invoke-Stage {
    param(
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)][int]$Ordinal,
        [string[]]$ComposeOverrides = @(),
        [System.Collections.IDictionary]$EnvironmentOverrides = @{},
        [Parameter(Mandatory)][scriptblock]$TestCommand
    )

    $runtime = New-StageRuntime -Stage $Name -Ordinal $Ordinal -ComposeOverrides $ComposeOverrides -EnvironmentOverrides $EnvironmentOverrides
    $environmentSnapshot = @{}
    $stageSucceeded = $false

    try {
        Prepare-StageEnvironment -Runtime $runtime -Snapshot $environmentSnapshot
        Start-Stage -Runtime $runtime
        Set-TestEndpoints -Runtime $runtime -Snapshot $environmentSnapshot
        & $TestCommand $runtime
        $stageSucceeded = $true
    } catch {
        Save-StageLogs -Runtime $runtime
        throw
    } finally {
        Stop-Stage -Runtime $runtime -KeepLogs:(-not $stageSucceeded)
        Restore-Environment -Snapshot $environmentSnapshot
        Remove-Item -LiteralPath $runtime.EnvFile -Force -ErrorAction SilentlyContinue
    }
}

function Invoke-MigrationConcurrentStage {
    param(
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)][int]$Ordinal,
        [Parameter(Mandatory)][string]$ComposeOverride
    )

    $runtime = New-StageRuntime -Stage $Name -Ordinal $Ordinal -ComposeOverrides @($ComposeOverride)
    $environmentSnapshot = @{}
    $stageSucceeded = $false

    try {
        Prepare-StageEnvironment -Runtime $runtime -Snapshot $environmentSnapshot
        Start-MigrationPrerequisites -Runtime $runtime

        $metadataBefore = @(Invoke-MySQLQuery -Runtime $runtime -Query "SHOW TABLES LIKE 'schema_migrations'")
        if ($metadataBefore.Count -ne 0) {
            throw "旧 schema fixture 在迁移前已包含 schema_migrations，无法验证自动升级：$($metadataBefore -join ', ')"
        }

        # 两个 job 共用同一隔离 MySQL；无论抢锁顺序如何，每个编号版本只能留下一个成功执行记录。
        Invoke-ConcurrentMigrateTasks -Runtime $runtime
        Assert-MigrationState -Runtime $runtime
        $runsAfterConcurrentMigrate = @(Get-MigrationRunSnapshot -Runtime $runtime)

        Start-ApiAndWebAfterMigration -Runtime $runtime
        Set-TestEndpoints -Runtime $runtime -Snapshot $environmentSnapshot
        Assert-ApiMigrationHealth -Runtime $runtime
        Assert-MigrationState -Runtime $runtime
        $runsAfterComposeDependency = @(Get-MigrationRunSnapshot -Runtime $runtime)
        if (($runsAfterConcurrentMigrate -join "`n") -ne ($runsAfterComposeDependency -join "`n")) {
            throw "Compose api 依赖的重复 migrate job 意外新增了版本执行记录。"
        }

        $stageSucceeded = $true
    } catch {
        Save-StageLogs -Runtime $runtime
        throw
    } finally {
        Stop-Stage -Runtime $runtime -KeepLogs:(-not $stageSucceeded)
        Restore-Environment -Snapshot $environmentSnapshot
        Remove-Item -LiteralPath $runtime.EnvFile -Force -ErrorAction SilentlyContinue
    }
}

function Invoke-RedisWrongPasswordStage {
    param(
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)][int]$Ordinal,
        [Parameter(Mandatory)][string]$ComposeOverride
    )

    $redisPassword = New-TestSecret
    $apiPassword = New-TestSecret
    if ($apiPassword -eq $redisPassword) {
        throw "无法生成不同的 Redis 正确密码和 API 错误密码。"
    }
    $runtime = New-StageRuntime -Stage $Name -Ordinal $Ordinal -ComposeOverrides @($ComposeOverride) -EnvironmentOverrides @{
        APP_REDIS_PASSWORD     = $redisPassword
        E2E_API_REDIS_PASSWORD = $apiPassword
    }
    $environmentSnapshot = @{}
    $stageSucceeded = $false

    try {
        Prepare-StageEnvironment -Runtime $runtime -Snapshot $environmentSnapshot
        # 仅启动 API 及其依赖；不会拉起 Web，也不会接触当前开发 Compose 项目。
        Invoke-Compose -Runtime $runtime -Arguments @("up", "--detach", "--build", "api")
        Initialize-DatabaseEndpoints -Runtime $runtime
        Wait-ForContainerHealthy -ContainerID $runtime.MySQLContainerId -Service "mysql"
        Wait-ForContainerHealthy -ContainerID $runtime.RedisContainerId -Service "redis"
        Assert-RedisPasswordMode -Runtime $runtime

        $apiContainerIds = @(Wait-ForComposeContainerIds -Runtime $runtime -Service "api" -IncludeStopped)
        if ($apiContainerIds.Count -ne 1) {
            throw "Redis 错误密码阶段 API 容器数 = $($apiContainerIds.Count)，期望 1。"
        }
        $apiExit = Wait-ForContainerExit -ContainerID $apiContainerIds[0] -Service "api"
        if ($apiExit.ExitCode -eq 0) {
            throw "APP_REDIS_PASSWORD 错误时 API 意外成功退出，未保持启动期 fail-fast。"
        }

        $apiLogs = Get-ComposeServiceLogs -Runtime $runtime -Service "api"
        if ($apiLogs -notmatch "(?i)(redis.*(authentication|auth)|wrongpass|noauth|invalid username-password)") {
            throw "APP_REDIS_PASSWORD 错误时 API 日志未包含明确 Redis 认证错误：$apiLogs"
        }

        $stageSucceeded = $true
    } catch {
        Save-StageLogs -Runtime $runtime
        throw
    } finally {
        Stop-Stage -Runtime $runtime -KeepLogs:(-not $stageSucceeded)
        Restore-Environment -Snapshot $environmentSnapshot
        Remove-Item -LiteralPath $runtime.EnvFile -Force -ErrorAction SilentlyContinue
    }
}

function Invoke-GoIntegrationTests {
    param([Parameter(Mandatory)][string]$Pattern)

    & go test -tags integration ./test/integration -count=1 -run $Pattern
    if ($LASTEXITCODE -ne 0) {
        throw "Go 集成测试 $Pattern 失败（退出码 $LASTEXITCODE）。"
    }
}

function Invoke-BrowserIntegrationTests {
    param(
        [Parameter(Mandatory)][string]$Project
    )

    if ($env:E2E_SKIP_BROWSER_INSTALL -ne "1") {
        & pnpm --dir frontend exec playwright install chromium webkit
        if ($LASTEXITCODE -ne 0) {
            throw "Playwright Chromium/WebKit 安装失败（退出码 $LASTEXITCODE）。"
        }
    }

    # pnpm 会将分隔符 `--` 原样传给脚本；这里直接传递项目参数，确保只运行当前隔离阶段对应的浏览器项目。
    & pnpm --dir frontend test:e2e --project $Project
    if ($LASTEXITCODE -ne 0) {
        throw "前端 E2E 测试（项目 $Project）失败（退出码 $LASTEXITCODE）。"
    }
}

try {
    Assert-Command -Name "docker"
    Assert-Command -Name "go"
    Assert-Command -Name "pnpm"
    New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null
    if (-not [string]::IsNullOrWhiteSpace($env:GITHUB_OUTPUT)) {
        Add-Content -LiteralPath $env:GITHUB_OUTPUT -Value "artifact_dir=$artifactRoot" -Encoding utf8
    }

    Push-Location $repoRoot
    $legacyEE23045Compose = Join-Path $repoRoot "deploy\test\docker-compose.migration-legacy-ee23045.yml"
    $legacy09FD516Compose = Join-Path $repoRoot "deploy\test\docker-compose.migration-legacy-09fd516.yml"
    $redisWrongPasswordCompose = Join-Path $repoRoot "deploy\test\docker-compose.redis-wrong-password.yml"
    foreach ($composeOverride in @($legacyEE23045Compose, $legacy09FD516Compose, $redisWrongPasswordCompose)) {
        if (-not (Test-Path -LiteralPath $composeOverride -PathType Leaf)) {
            throw "缺少集成测试 Compose 覆盖文件：$composeOverride"
        }
    }

    # 空密码：空库启动时 API 必须等待 migrate 成功，Redis 仍明确允许无认证探测。
    Invoke-Stage -Name "http" -Ordinal 1 -TestCommand {
        param($runtime)
        Assert-RedisPasswordMode -Runtime $runtime
        Assert-MigrationState -Runtime $runtime
        Assert-ApiMigrationHealth -Runtime $runtime
        Invoke-GoIntegrationTests -Pattern "^TestCore"
    }
    # 两份已固定历史 schema 都通过 Compose 的 migrate 依赖链自动升级，而非由测试直接导入 SQL。
    Invoke-Stage -Name "migration-legacy-ee23045" -Ordinal 2 -ComposeOverrides @($legacyEE23045Compose) -TestCommand {
        param($runtime)
        Assert-MigrationState -Runtime $runtime
        Assert-ApiMigrationHealth -Runtime $runtime
    }
    Invoke-Stage -Name "migration-legacy-09fd516" -Ordinal 3 -ComposeOverrides @($legacy09FD516Compose) -TestCommand {
        param($runtime)
        Assert-MigrationState -Runtime $runtime
        $runsBeforeRepeat = @(Get-MigrationRunSnapshot -Runtime $runtime)
        Invoke-MigrateTask -Runtime $runtime
        Assert-MigrationState -Runtime $runtime
        $runsAfterRepeat = @(Get-MigrationRunSnapshot -Runtime $runtime)
        if (($runsBeforeRepeat -join "`n") -ne ($runsAfterRepeat -join "`n")) {
            throw "已完成的 migrate task 重复运行后意外新增了版本执行记录。"
        }
        Assert-ApiMigrationHealth -Runtime $runtime
    }
    Invoke-MigrationConcurrentStage -Name "migration-concurrent-09fd516" -Ordinal 4 -ComposeOverride $legacy09FD516Compose

    # 正确非空密码：Redis 必须拒绝匿名访问、接受正确认证，且 API 在认证 Redis 重启后恢复健康。
    Invoke-Stage -Name "redis-auth" -Ordinal 5 -EnvironmentOverrides @{ APP_REDIS_PASSWORD = (New-TestSecret) } -TestCommand {
        param($runtime)
        Assert-RedisPasswordMode -Runtime $runtime
        Assert-ApiMigrationHealth -Runtime $runtime
        Invoke-GoIntegrationTests -Pattern "^TestExtendedRedisFailClosed$"
    }

    # 错误密码：API 必须在启动期 fail-fast，且输出可定位的 Redis 认证错误；该阶段不会启动 Web。
    Invoke-RedisWrongPasswordStage -Name "redis-wrong-password" -Ordinal 6 -ComposeOverride $redisWrongPasswordCompose

    $browserProjects = @("chromium", "mobile-chromium", "mobile-webkit")
    $browserOrdinal = 7
    foreach ($browserProject in $browserProjects) {
        $projectName = $browserProject
        Invoke-Stage -Name "browser-$projectName" -Ordinal $browserOrdinal -TestCommand {
            param($runtime)
            Invoke-BrowserIntegrationTests -Project $projectName
        }
        $browserOrdinal++
    }
    if ($Suite -eq "extended") {
        Invoke-Stage -Name "extended" -Ordinal 10 -TestCommand {
            # 密码保护 Redis 的停止/重启恢复已在 redis-auth 核心阶段覆盖；此处保留其余扩展故障注入。
            Invoke-GoIntegrationTests -Pattern "^TestExtended(ConcurrentRegistrationAndRefreshRotation|BackupDatabaseStageFailure)$"
        }
    }

    $runSucceeded = $true
    Write-Host "$Suite 集成测试通过。"
} finally {
    Pop-Location -ErrorAction SilentlyContinue
    if ($runSucceeded) {
        # 成功运行不保留日志、临时环境文件或测试凭据。
        Remove-Item -LiteralPath $artifactRoot -Recurse -Force -ErrorAction SilentlyContinue
    } else {
        Write-Warning "集成测试失败，已保留无凭据日志：$artifactRoot"
    }
}
