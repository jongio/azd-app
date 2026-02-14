using Azure.Core;
using Azure.Identity;
using Azure.ResourceManager;
using Azure.ResourceManager.Resources;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

// Configure DefaultAzureCredential to only use AzureDeveloperCliCredential
// This ensures we're testing the shim, not some other credential source.
var credentialOptions = new DefaultAzureCredentialOptions
{
    ExcludeEnvironmentCredential = true,
    ExcludeManagedIdentityCredential = true,
    ExcludeVisualStudioCredential = true,
    ExcludeAzureCliCredential = true,
    ExcludeSharedTokenCacheCredential = true,
    ExcludeInteractiveBrowserCredential = true,
    ExcludeAzurePowerShellCredential = true,
    ExcludeWorkloadIdentityCredential = true,
    ExcludeAzureDeveloperCliCredential = false
};

var credential = new DefaultAzureCredential(credentialOptions);

app.Logger.LogInformation("=== Container Auth Test App ===");
app.Logger.LogInformation("DefaultAzureCredential configured with AzureDeveloperCliCredential only");
app.Logger.LogInformation("AZD_AUTH_HOST={Host}", Environment.GetEnvironmentVariable("AZD_AUTH_HOST") ?? "(not set)");
app.Logger.LogInformation("AZD_AUTH_PORT={Port}", Environment.GetEnvironmentVariable("AZD_AUTH_PORT") ?? "(not set)");
app.Logger.LogInformation("AZD_AUTH_CERTS_DIR={Dir}", Environment.GetEnvironmentVariable("AZD_AUTH_CERTS_DIR") ?? "(not set)");

app.MapGet("/health", () =>
{
    return Results.Ok(new { status = "ok", timestamp = DateTime.UtcNow });
});

// Validates the shim is mounted and certs are present (no Azure call needed)
app.MapGet("/check", (ILogger<Program> logger) =>
{
    var shimExists = System.IO.File.Exists("/usr/local/bin/azd");
    var certsDir = Environment.GetEnvironmentVariable("AZD_AUTH_CERTS_DIR") ?? "/run/secrets/azd-auth";
    var caExists = System.IO.File.Exists(System.IO.Path.Combine(certsDir, "ca.pem"));
    var clientCertExists = System.IO.File.Exists(System.IO.Path.Combine(certsDir, "client.pem"));
    var clientKeyExists = System.IO.File.Exists(System.IO.Path.Combine(certsDir, "client-key.pem"));
    var authHost = Environment.GetEnvironmentVariable("AZD_AUTH_HOST");
    var authPort = Environment.GetEnvironmentVariable("AZD_AUTH_PORT");

    var allOk = shimExists && caExists && clientCertExists && clientKeyExists
                && !string.IsNullOrEmpty(authHost) && !string.IsNullOrEmpty(authPort);

    logger.LogInformation("Check: shim={Shim} ca={CA} cert={Cert} key={Key} host={Host} port={Port}",
        shimExists, caExists, clientCertExists, clientKeyExists, authHost, authPort);

    return Results.Ok(new
    {
        success = allOk,
        shimMounted = shimExists,
        caCertPresent = caExists,
        clientCertPresent = clientCertExists,
        clientKeyPresent = clientKeyExists,
        authHost = authHost ?? "(not set)",
        authPort = authPort ?? "(not set)"
    });
});

// Acquires a token via DefaultAzureCredential (exercises full shim → mTLS → host chain)
app.MapGet("/token", async (ILogger<Program> logger) =>
{
    try
    {
        logger.LogInformation("Acquiring token for https://management.azure.com/.default ...");

        var tokenRequestContext = new TokenRequestContext(
            scopes: new[] { "https://management.azure.com/.default" }
        );

        var startTime = DateTime.UtcNow;
        var token = await credential.GetTokenAsync(tokenRequestContext, CancellationToken.None);
        var duration = DateTime.UtcNow - startTime;

        logger.LogInformation("Token acquired in {Duration}ms", duration.TotalMilliseconds);

        return Results.Ok(new
        {
            success = true,
            tokenPrefix = token.Token[..20] + "...",
            expiresOn = token.ExpiresOn,
            durationMs = duration.TotalMilliseconds
        });
    }
    catch (Exception ex)
    {
        logger.LogError(ex, "Token acquisition failed");
        return Results.Json(new
        {
            success = false,
            error = ex.GetType().Name,
            message = ex.Message
        }, statusCode: 500);
    }
});

// Lists Azure resource groups (proves real Azure API works from container)
app.MapGet("/azure", async (ILogger<Program> logger) =>
{
    try
    {
        logger.LogInformation("Creating ArmClient...");
        var armClient = new ArmClient(credential);

        var subscription = await armClient.GetDefaultSubscriptionAsync();
        logger.LogInformation("Subscription: {Name}", subscription.Data.DisplayName);

        var rgNames = new List<string>();
        await foreach (var rg in subscription.GetResourceGroups().GetAllAsync())
        {
            rgNames.Add(rg.Data.Name);
        }

        logger.LogInformation("Found {Count} resource group(s)", rgNames.Count);

        return Results.Ok(new
        {
            success = true,
            subscription = subscription.Data.DisplayName,
            resourceGroups = rgNames,
            count = rgNames.Count
        });
    }
    catch (Exception ex)
    {
        logger.LogError(ex, "Azure API call failed");
        return Results.Json(new
        {
            success = false,
            error = ex.GetType().Name,
            message = ex.Message
        }, statusCode: 500);
    }
});

app.Logger.LogInformation("Endpoints: GET /health, GET /check, GET /token, GET /azure");
app.Run();
