using System;

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

builder.Build().Run();

