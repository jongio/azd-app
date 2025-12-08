using DotnetApi;

namespace DotnetApi.Tests;

[Trait("Category", "Unit")]
public class CalculatorUnitTests
{
    private readonly Calculator _calculator = new();

    [Fact]
    public void Add_PositiveNumbers_ReturnsSum()
    {
        Assert.Equal(5, _calculator.Add(2, 3));
    }

    [Fact]
    public void Add_NegativeNumbers_ReturnsSum()
    {
        Assert.Equal(-5, _calculator.Add(-2, -3));
    }

    [Fact]
    public void Add_WithZero_ReturnsOther()
    {
        Assert.Equal(5, _calculator.Add(5, 0));
    }

    [Fact]
    public void Add_Floats_ReturnsSum()
    {
        Assert.Equal(4.0, _calculator.Add(1.5, 2.5));
    }

    [Fact]
    public void Subtract_PositiveResult_ReturnsCorrect()
    {
        Assert.Equal(2, _calculator.Subtract(5, 3));
    }

    [Fact]
    public void Subtract_NegativeResult_ReturnsCorrect()
    {
        Assert.Equal(-2, _calculator.Subtract(3, 5));
    }

    [Fact]
    public void Multiply_PositiveNumbers_ReturnsProduct()
    {
        Assert.Equal(12, _calculator.Multiply(3, 4));
    }

    [Fact]
    public void Multiply_WithZero_ReturnsZero()
    {
        Assert.Equal(0, _calculator.Multiply(5, 0));
    }

    [Fact]
    public void Multiply_NegativeNumbers_ReturnsProduct()
    {
        Assert.Equal(-12, _calculator.Multiply(-3, 4));
    }

    [Fact]
    public void Divide_EvenDivision_ReturnsQuotient()
    {
        Assert.Equal(5, _calculator.Divide(10, 2));
    }

    [Fact]
    public void Divide_DecimalResult_ReturnsQuotient()
    {
        Assert.Equal(2.5, _calculator.Divide(10, 4));
    }

    [Fact]
    public void Divide_ByZero_ThrowsException()
    {
        Assert.Throws<DivideByZeroException>(() => _calculator.Divide(10, 0));
    }

    [Fact]
    public void Factorial_Of5_Returns120()
    {
        Assert.Equal(120, _calculator.Factorial(5));
    }

    [Fact]
    public void Factorial_Of0_Returns1()
    {
        Assert.Equal(1, _calculator.Factorial(0));
    }

    [Fact]
    public void Factorial_Of1_Returns1()
    {
        Assert.Equal(1, _calculator.Factorial(1));
    }

    [Fact]
    public void Factorial_Negative_ThrowsException()
    {
        Assert.Throws<ArgumentException>(() => _calculator.Factorial(-1));
    }

    [Fact]
    public void Fibonacci_Of0_Returns0()
    {
        Assert.Equal(0, _calculator.Fibonacci(0));
    }

    [Fact]
    public void Fibonacci_Of1_Returns1()
    {
        Assert.Equal(1, _calculator.Fibonacci(1));
    }

    [Fact]
    public void Fibonacci_Of10_Returns55()
    {
        Assert.Equal(55, _calculator.Fibonacci(10));
    }

    [Fact]
    public void Fibonacci_Negative_ThrowsException()
    {
        Assert.Throws<ArgumentException>(() => _calculator.Fibonacci(-1));
    }

    [Fact]
    public void IsPrime_2_ReturnsTrue()
    {
        Assert.True(_calculator.IsPrime(2));
    }

    [Fact]
    public void IsPrime_7_ReturnsTrue()
    {
        Assert.True(_calculator.IsPrime(7));
    }

    [Fact]
    public void IsPrime_4_ReturnsFalse()
    {
        Assert.False(_calculator.IsPrime(4));
    }

    [Fact]
    public void IsPrime_1_ReturnsFalse()
    {
        Assert.False(_calculator.IsPrime(1));
    }

    [Fact]
    public void IsPrime_Negative_ReturnsFalse()
    {
        Assert.False(_calculator.IsPrime(-5));
    }

    [Fact]
    public void Power_2To10_Returns1024()
    {
        Assert.Equal(1024, _calculator.Power(2, 10));
    }

    [Fact]
    public void Abs_Negative_ReturnsPositive()
    {
        Assert.Equal(5, _calculator.Abs(-5));
    }

    [Fact]
    public void Abs_Positive_ReturnsSame()
    {
        Assert.Equal(5, _calculator.Abs(5));
    }
}
