var builder = WebApplication.CreateBuilder(args);

// Add services
builder.Services.AddSingleton<DotnetApi.Calculator>();

var app = builder.Build();

// Health endpoint
app.MapGet("/health", () => new { status = "healthy" });

// Calculate endpoint
app.MapPost("/calculate", (CalculateRequest request, DotnetApi.Calculator calculator) =>
{
    try
    {
        double result = request.Operation switch
        {
            "add" => calculator.Add(request.A, request.B),
            "subtract" => calculator.Subtract(request.A, request.B),
            "multiply" => calculator.Multiply(request.A, request.B),
            "divide" => calculator.Divide(request.A, request.B),
            _ => throw new ArgumentException($"Unknown operation: {request.Operation}")
        };
        return Results.Ok(new { result });
    }
    catch (Exception ex)
    {
        return Results.BadRequest(new { error = ex.Message });
    }
});

app.Run();

public record CalculateRequest(string Operation, double A, double B);

// Make Program accessible for testing
public partial class Program { }
