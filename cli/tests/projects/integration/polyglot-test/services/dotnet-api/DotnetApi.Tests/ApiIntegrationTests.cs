using System.Net;
using System.Net.Http.Json;
using Microsoft.AspNetCore.Mvc.Testing;

namespace DotnetApi.Tests;

[Trait("Category", "Integration")]
public class ApiIntegrationTests : IClassFixture<WebApplicationFactory<Program>>
{
    private readonly WebApplicationFactory<Program> _factory;
    private readonly HttpClient _client;

    public ApiIntegrationTests(WebApplicationFactory<Program> factory)
    {
        _factory = factory;
        _client = _factory.CreateClient();
    }

    [Fact]
    public async Task Health_ReturnsHealthy()
    {
        var response = await _client.GetAsync("/health");
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        
        var content = await response.Content.ReadFromJsonAsync<HealthResponse>();
        Assert.Equal("healthy", content?.Status);
    }

    [Fact]
    public async Task Calculate_Add_ReturnsCorrectResult()
    {
        var request = new { operation = "add", a = 5, b = 3 };
        var response = await _client.PostAsJsonAsync("/calculate", request);
        
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        var content = await response.Content.ReadFromJsonAsync<CalculateResponse>();
        Assert.Equal(8, content?.Result);
    }

    [Fact]
    public async Task Calculate_Subtract_ReturnsCorrectResult()
    {
        var request = new { operation = "subtract", a = 10, b = 4 };
        var response = await _client.PostAsJsonAsync("/calculate", request);
        
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        var content = await response.Content.ReadFromJsonAsync<CalculateResponse>();
        Assert.Equal(6, content?.Result);
    }

    [Fact]
    public async Task Calculate_Multiply_ReturnsCorrectResult()
    {
        var request = new { operation = "multiply", a = 3, b = 7 };
        var response = await _client.PostAsJsonAsync("/calculate", request);
        
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        var content = await response.Content.ReadFromJsonAsync<CalculateResponse>();
        Assert.Equal(21, content?.Result);
    }

    [Fact]
    public async Task Calculate_Divide_ReturnsCorrectResult()
    {
        var request = new { operation = "divide", a = 20, b = 4 };
        var response = await _client.PostAsJsonAsync("/calculate", request);
        
        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        var content = await response.Content.ReadFromJsonAsync<CalculateResponse>();
        Assert.Equal(5, content?.Result);
    }

    [Fact]
    public async Task Calculate_DivideByZero_ReturnsBadRequest()
    {
        var request = new { operation = "divide", a = 10, b = 0 };
        var response = await _client.PostAsJsonAsync("/calculate", request);
        
        Assert.Equal(HttpStatusCode.BadRequest, response.StatusCode);
    }

    [Fact]
    public async Task Calculate_UnknownOperation_ReturnsBadRequest()
    {
        var request = new { operation = "unknown", a = 1, b = 2 };
        var response = await _client.PostAsJsonAsync("/calculate", request);
        
        Assert.Equal(HttpStatusCode.BadRequest, response.StatusCode);
    }

    private record HealthResponse(string Status);
    private record CalculateResponse(double Result, string? Error);
}
