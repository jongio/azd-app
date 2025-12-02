using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Http;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

app.MapGet("/", () => "Dotnet Tool - HTTP service with watch mode");
app.MapGet("/health", () => Results.Json(new { status = "healthy", service = "dotnet-watch" }));

app.Run("http://localhost:5002");
