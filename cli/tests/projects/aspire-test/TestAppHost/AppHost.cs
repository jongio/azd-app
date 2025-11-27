using System;
using System.Threading.Tasks;

var builder = DistributedApplication.CreateBuilder(args);

// Display azd environment variables to verify they're being passed through
Console.WriteLine("========================================");
Console.WriteLine("🔍 Checking azd Environment Variables:");
Console.WriteLine("========================================");

var azdServer = Environment.GetEnvironmentVariable("AZD_SERVER");
var azdAccessToken = Environment.GetEnvironmentVariable("AZD_ACCESS_TOKEN");
var azureSubscription = Environment.GetEnvironmentVariable("AZURE_SUBSCRIPTION_ID");
var azureEnvName = Environment.GetEnvironmentVariable("AZURE_ENV_NAME");

Console.WriteLine($"AZD_SERVER: {(string.IsNullOrEmpty(azdServer) ? "❌ NOT SET" : $"✅ {azdServer}")}");
Console.WriteLine($"AZD_ACCESS_TOKEN: {(string.IsNullOrEmpty(azdAccessToken) ? "❌ NOT SET" : $"✅ {azdAccessToken[..Math.Min(20, azdAccessToken.Length)]}...")}");
Console.WriteLine($"AZURE_SUBSCRIPTION_ID: {(string.IsNullOrEmpty(azureSubscription) ? "❌ NOT SET" : $"✅ {azureSubscription}")}");
Console.WriteLine($"AZURE_ENV_NAME: {(string.IsNullOrEmpty(azureEnvName) ? "❌ NOT SET" : $"✅ {azureEnvName}")}");

// List all environment variables that start with AZD_ or AZURE_
Console.WriteLine("\n📋 All AZD/AZURE Environment Variables:");
Console.WriteLine("----------------------------------------");
var envVars = Environment.GetEnvironmentVariables();
var foundAny = false;
foreach (var key in envVars.Keys)
{
    var keyStr = key.ToString() ?? "";
    if (keyStr.StartsWith("AZD_", StringComparison.OrdinalIgnoreCase) || 
        keyStr.StartsWith("AZURE_", StringComparison.OrdinalIgnoreCase))
    {
        foundAny = true;
        var value = envVars[key]?.ToString() ?? "";
        // Truncate long values for security/readability
        var displayValue = value.Length > 50 ? value[..50] + "..." : value;
        Console.WriteLine($"  {keyStr} = {displayValue}");
    }
}

if (!foundAny)
{
    Console.WriteLine("  ⚠️  No AZD_ or AZURE_ environment variables found!");
}

Console.WriteLine("========================================\n");

// Service 1: API Service (simulates HTTP API with request logging)
var api = builder.AddContainer("api", "alpine", "latest")
    .WithArgs("sh", "-c", @"
        echo '🚀 API Service starting...';
        i=0;
        while true; do
            i=$((i+1));
            echo ""[INFO] $(date +'%H:%M:%S') - Processing request #$i"";
            sleep 2;
            if [ $((i % 5)) -eq 0 ]; then
                echo ""[WARN] $(date +'%H:%M:%S') - High latency detected on /api/users endpoint"";
            fi;
            if [ $((i % 10)) -eq 0 ]; then
                echo ""[ERROR] $(date +'%H:%M:%S') - Failed to connect to database - connection timeout"";
            fi;
        done
    ");

// Service 2: Web Service (simulates frontend build/serve logs)
var web = builder.AddContainer("web", "alpine", "latest")
    .WithArgs("sh", "-c", @"
        echo '🌐 Web Service starting...';
        echo '[INFO] Building React application...';
        echo '[INFO] webpack 5.88.0 compiled successfully';
        i=0;
        while true; do
            i=$((i+1));
            echo ""[INFO] $(date +'%H:%M:%S') - Serving /dashboard to 192.168.1.$((i % 255))"";
            sleep 3;
            if [ $((i % 7)) -eq 0 ]; then
                echo ""[WARN] $(date +'%H:%M:%S') - Deprecated API usage in component/UserList.tsx"";
            fi;
            if [ $((i % 15)) -eq 0 ]; then
                echo ""[ERROR] $(date +'%H:%M:%S') - Failed to load chunk vendors~main.js"";
            fi;
        done
    ");

// Service 3: Worker Service (simulates background job processing)
var worker = builder.AddContainer("worker", "alpine", "latest")
    .WithArgs("sh", "-c", @"
        echo '⚙️ Worker Service starting...';
        echo '[INFO] Connected to job queue';
        i=0;
        while true; do
            i=$((i+1));
            echo ""[INFO] $(date +'%H:%M:%S') - Processing job batch #$i (5 jobs)"";
            sleep 4;
            echo ""[INFO] $(date +'%H:%M:%S') - Completed job #$i successfully"";
            if [ $((i % 6)) -eq 0 ]; then
                echo ""[WARN] $(date +'%H:%M:%S') - Job retry count exceeded for task email_notification_$i"";
            fi;
            if [ $((i % 12)) -eq 0 ]; then
                echo ""[ERROR] $(date +'%H:%M:%S') - Job failed - unable to process image resize task"";
            fi;
        done
    ");

// Service 4: Cache Service (simulates Redis-style cache operations)
var cache = builder.AddContainer("cache", "alpine", "latest")
    .WithArgs("sh", "-c", @"
        echo '💾 Cache Service starting...';
        echo '[INFO] Redis server initialized on port 6379';
        echo '[INFO] 0 keys loaded from disk';
        i=0;
        while true; do
            i=$((i+1));
            ops=$((i % 4));
            case $ops in
                0) echo ""[INFO] $(date +'%H:%M:%S') - GET user:session:$i (hit)"" ;;
                1) echo ""[INFO] $(date +'%H:%M:%S') - SET product:$i (ttl: 3600s)"" ;;
                2) echo ""[INFO] $(date +'%H:%M:%S') - DEL expired_key_$i"" ;;
                3) echo ""[INFO] $(date +'%H:%M:%S') - INCR counter:views:$i"" ;;
            esac;
            sleep 1;
            if [ $((i % 20)) -eq 0 ]; then
                echo ""[WARN] $(date +'%H:%M:%S') - Memory usage: 85% - consider eviction"";
            fi;
            if [ $((i % 30)) -eq 0 ]; then
                echo ""[ERROR] $(date +'%H:%M:%S') - Connection refused from client 192.168.1.100"";
            fi;
        done
    ");

// Service 5: Database Service (simulates SQL query logs)
var database = builder.AddContainer("database", "alpine", "latest")
    .WithArgs("sh", "-c", @"
        echo '🗄️ Database Service starting...';
        echo '[INFO] PostgreSQL 15.3 starting on port 5432';
        echo '[INFO] Database system ready to accept connections';
        i=0;
        while true; do
            i=$((i+1));
            queries=('SELECT * FROM users WHERE id' 'INSERT INTO orders' 'UPDATE products SET' 'DELETE FROM sessions WHERE');
            query=${queries[$((i % 4))]};
            echo ""[INFO] $(date +'%H:%M:%S') - $query=$i (duration: $((i % 50))ms)"";
            sleep 2;
            if [ $((i % 8)) -eq 0 ]; then
                echo ""[WARN] $(date +'%H:%M:%S') - Slow query detected: SELECT * FROM logs (2.5s)"";
            fi;
            if [ $((i % 20)) -eq 0 ]; then
                echo ""[ERROR] $(date +'%H:%M:%S') - Deadlock detected: transaction rolled back"";
            fi;
        done
    ");

builder.Build().Run();
