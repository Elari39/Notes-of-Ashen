[CmdletBinding()]
param(
    [ValidateSet("core", "extended")]
    [string]$Suite = "core"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$composeFiles = @(
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

    return @(
        "compose",
        "--env-file", $Runtime.EnvFile,
        "--project-name", $Runtime.Project,
        "--file", $composeFiles[0],
        "--file", $composeFiles[1]
    )
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

function Get-ComposeContainerId {
    param(
        [Parameter(Mandatory)]$Runtime,
        [Parameter(Mandatory)][string]$Service
    )

    $composeArguments = Get-ComposeArguments -Runtime $Runtime
    $containerLines = @(& docker @composeArguments ps --quiet $Service)
    $containerId = @($containerLines | Select-Object -First 1)
    if ($LASTEXITCODE -ne 0 -or $containerId.Count -eq 0 -or [string]::IsNullOrWhiteSpace([string]$containerId[0])) {
        throw "无法获取 $Service 容器 ID。"
    }

    return ([string]$containerId[0]).Trim()
}

function New-StageRuntime {
    param(
        [Parameter(Mandatory)][string]$Stage,
        [Parameter(Mandatory)][int]$Ordinal
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
        APP_RABBITMQ_PASSWORD        = (New-TestSecret)
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
        MySQLRootPassword  = $mysqlRootPassword
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

function Start-Stage {
    param([Parameter(Mandatory)]$Runtime)

    Invoke-Compose -Runtime $Runtime -Arguments @("up", "--detach", "--build", "web", "api", "mysql", "redis")
    $Runtime | Add-Member -NotePropertyName WebPort -NotePropertyValue (Get-ComposePort -Runtime $Runtime -Service "web" -ContainerPort 8080)
    $Runtime | Add-Member -NotePropertyName ApiPort -NotePropertyValue (Get-ComposePort -Runtime $Runtime -Service "api" -ContainerPort 19000)
    $Runtime | Add-Member -NotePropertyName MySQLPort -NotePropertyValue (Get-ComposePort -Runtime $Runtime -Service "mysql" -ContainerPort 3306)
    $Runtime | Add-Member -NotePropertyName RedisPort -NotePropertyValue (Get-ComposePort -Runtime $Runtime -Service "redis" -ContainerPort 6379)
    $Runtime | Add-Member -NotePropertyName WebBaseUrl -NotePropertyValue "http://127.0.0.1:$($Runtime.WebPort)"
    $Runtime | Add-Member -NotePropertyName ApiBaseUrl -NotePropertyValue "http://127.0.0.1:$($Runtime.ApiPort)"
    $Runtime | Add-Member -NotePropertyName RedisContainerId -NotePropertyValue (Get-ComposeContainerId -Runtime $Runtime -Service "redis")
    $Runtime | Add-Member -NotePropertyName MySQLContainerId -NotePropertyValue (Get-ComposeContainerId -Runtime $Runtime -Service "mysql")

    Wait-ForApi -Runtime $Runtime
    Wait-ForWeb -Runtime $Runtime
}

function Set-TestEndpoints {
    param(
        [Parameter(Mandatory)]$Runtime,
        [Parameter(Mandatory)][hashtable]$Snapshot
    )

    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_WEB_BASE_URL" -Value $Runtime.WebBaseUrl
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_API_BASE_URL" -Value $Runtime.ApiBaseUrl
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_REDIS_URL" -Value "redis://127.0.0.1:$($Runtime.RedisPort)/0"
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_MYSQL_DSN" -Value "notes_test:$($Runtime.ComposeEnvironment['APP_MYSQL_PASSWORD'])@tcp(127.0.0.1:$($Runtime.MySQLPort))/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local"
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_MYSQL_ROOT_DSN" -Value "root:$($Runtime.MySQLRootPassword)@tcp(127.0.0.1:$($Runtime.MySQLPort))/notes_of_ashen?charset=utf8mb4&parseTime=true&loc=Local"
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_REDIS_CONTAINER_ID" -Value $Runtime.RedisContainerId
    Set-EnvironmentValue -Snapshot $Snapshot -Name "E2E_MYSQL_CONTAINER_ID" -Value $Runtime.MySQLContainerId
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
        [Parameter(Mandatory)][scriptblock]$TestCommand
    )

    $runtime = New-StageRuntime -Stage $Name -Ordinal $Ordinal
    $environmentSnapshot = @{}
    $stageSucceeded = $false

    try {
        Prepare-StageEnvironment -Runtime $runtime -Snapshot $environmentSnapshot
        Start-Stage -Runtime $runtime
        Set-TestEndpoints -Runtime $runtime -Snapshot $environmentSnapshot
        & $TestCommand
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
    if ($env:E2E_SKIP_BROWSER_INSTALL -ne "1") {
        & pnpm --dir frontend exec playwright install chromium
        if ($LASTEXITCODE -ne 0) {
            throw "Playwright Chromium 安装失败（退出码 $LASTEXITCODE）。"
        }
    }

    & pnpm --dir frontend test:e2e
    if ($LASTEXITCODE -ne 0) {
        throw "前端 E2E 测试失败（退出码 $LASTEXITCODE）。"
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
    Invoke-Stage -Name "http" -Ordinal 1 -TestCommand { Invoke-GoIntegrationTests -Pattern "^TestCore" }
    Invoke-Stage -Name "browser" -Ordinal 2 -TestCommand { Invoke-BrowserIntegrationTests }
    if ($Suite -eq "extended") {
        Invoke-Stage -Name "extended" -Ordinal 3 -TestCommand {
            # Redis 停止/重启可能使 Windows Docker Desktop 的宿主端口映射延迟恢复。
            # 因此先完成仍依赖 E2E_REDIS_URL 的并发与备份故障注入，再将 Redis
            # fail-closed 用例置于本生命周期的最后，避免其影响后续断言。
            Invoke-GoIntegrationTests -Pattern "^TestExtended(ConcurrentRegistrationAndRefreshRotation|BackupDatabaseStageFailure)$"
            Invoke-GoIntegrationTests -Pattern "^TestExtendedRedisFailClosed$"
        }
    }

    $runSucceeded = $true
    Write-Host "P3-01 $Suite 集成测试通过。"
} finally {
    Pop-Location -ErrorAction SilentlyContinue
    if ($runSucceeded) {
        # 成功运行不保留日志、临时环境文件或测试凭据。
        Remove-Item -LiteralPath $artifactRoot -Recurse -Force -ErrorAction SilentlyContinue
    } else {
        Write-Warning "集成测试失败，已保留无凭据日志：$artifactRoot"
    }
}
