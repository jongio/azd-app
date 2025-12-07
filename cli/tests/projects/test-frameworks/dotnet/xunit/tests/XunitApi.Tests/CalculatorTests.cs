using Xunit;
using XunitApi;

namespace XunitApi.Tests;

public class CalculatorTests
{
    private readonly Calculator _calculator = new();

    [Fact]
    public void Add_ReturnsSum()
    {
        Assert.Equal(5, _calculator.Add(2, 3));
        Assert.Equal(0, _calculator.Add(-1, 1));
    }

    [Fact]
    public void Subtract_ReturnsDifference()
    {
        Assert.Equal(2, _calculator.Subtract(5, 3));
        Assert.Equal(-2, _calculator.Subtract(3, 5));
    }

    [Fact]
    public void Multiply_ReturnsProduct()
    {
        Assert.Equal(12, _calculator.Multiply(3, 4));
        Assert.Equal(0, _calculator.Multiply(0, 100));
    }

    [Fact]
    public void Divide_ReturnsQuotient()
    {
        Assert.Equal(5, _calculator.Divide(10, 2));
        Assert.Equal(3.5, _calculator.Divide(7, 2));
    }

    [Fact]
    public void Divide_ByZero_ThrowsException()
    {
        Assert.Throws<DivideByZeroException>(() => _calculator.Divide(10, 0));
    }

    [Theory]
    [InlineData(2, 3, 8)]
    [InlineData(5, 0, 1)]
    [InlineData(10, 2, 100)]
    public void Power_ReturnsCorrectResult(double baseNum, double exp, double expected)
    {
        Assert.Equal(expected, _calculator.Power(baseNum, exp));
    }

    [Fact]
    public void Sqrt_ReturnsSquareRoot()
    {
        Assert.Equal(4, _calculator.Sqrt(16));
        Assert.Equal(0, _calculator.Sqrt(0));
    }

    [Fact]
    public void Sqrt_Negative_ThrowsException()
    {
        Assert.Throws<ArgumentException>(() => _calculator.Sqrt(-1));
    }

    [Theory]
    [InlineData(0, 1)]
    [InlineData(1, 1)]
    [InlineData(5, 120)]
    [InlineData(10, 3628800)]
    public void Factorial_ReturnsCorrectResult(int n, int expected)
    {
        Assert.Equal(expected, _calculator.Factorial(n));
    }

    [Fact]
    public void Factorial_Negative_ThrowsException()
    {
        Assert.Throws<ArgumentException>(() => _calculator.Factorial(-1));
    }

    [Theory]
    [InlineData(0, 0)]
    [InlineData(1, 1)]
    [InlineData(2, 1)]
    [InlineData(10, 55)]
    public void Fibonacci_ReturnsCorrectResult(int n, int expected)
    {
        Assert.Equal(expected, _calculator.Fibonacci(n));
    }

    [Theory]
    [InlineData(2, true)]
    [InlineData(7, true)]
    [InlineData(13, true)]
    [InlineData(97, true)]
    public void IsPrime_True_ForPrimes(int n, bool expected)
    {
        Assert.Equal(expected, _calculator.IsPrime(n));
    }

    [Theory]
    [InlineData(0, false)]
    [InlineData(1, false)]
    [InlineData(4, false)]
    [InlineData(100, false)]
    public void IsPrime_False_ForNonPrimes(int n, bool expected)
    {
        Assert.Equal(expected, _calculator.IsPrime(n));
    }

    [Theory]
    [InlineData(12, 8, 4)]
    [InlineData(17, 13, 1)]
    [InlineData(100, 25, 25)]
    public void Gcd_ReturnsGreatestCommonDivisor(int a, int b, int expected)
    {
        Assert.Equal(expected, _calculator.Gcd(a, b));
    }

    [Theory]
    [InlineData(4, 6, 12)]
    [InlineData(3, 5, 15)]
    [InlineData(12, 18, 36)]
    public void Lcm_ReturnsLeastCommonMultiple(int a, int b, int expected)
    {
        Assert.Equal(expected, _calculator.Lcm(a, b));
    }

    [Fact]
    public void Modulo_ReturnsRemainder()
    {
        Assert.Equal(1, _calculator.Modulo(10, 3));
        Assert.Equal(0, _calculator.Modulo(15, 5));
    }

    [Fact]
    public void Modulo_ByZero_ThrowsException()
    {
        Assert.Throws<DivideByZeroException>(() => _calculator.Modulo(10, 0));
    }
}
