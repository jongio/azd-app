using System.Net;
using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
using Microsoft.Extensions.Logging;

namespace FunctionApp;

public class HttpTriggerFunction
{
    private readonly ILogger _logger;

    public HttpTriggerFunction(ILoggerFactory loggerFactory)
    {
        _logger = loggerFactory.CreateLogger<HttpTriggerFunction>();
    }

    [Function("HttpTrigger")]
    public HttpResponseData Run(
        [HttpTrigger(AuthorizationLevel.Anonymous, "get", "post")] HttpRequestData req)
    {
        _logger.LogInformation("C# HTTP trigger function processed a request (.NET Isolated).");

        var query = System.Web.HttpUtility.ParseQueryString(req.Url.Query);
        var name = query["name"] ?? "World";

        var response = req.CreateResponse(HttpStatusCode.OK);
        response.Headers.Add("Content-Type", "application/json");

        var responseBody = new
        {
            message = $"Hello, {name}! (.NET Isolated)",
            timestamp = DateTime.UtcNow.ToString("o"),
            method = req.Method
        };

        response.WriteAsJsonAsync(responseBody);

        return response;
    }
}
