namespace DotnetApi;

/// <summary>
/// Calculator service with basic math operations.
/// </summary>
public class Calculator
{
    /// <summary>
    /// Adds two numbers.
    /// </summary>
    public double Add(double a, double b) => a + b;

    /// <summary>
    /// Subtracts b from a.
    /// </summary>
    public double Subtract(double a, double b) => a - b;

    /// <summary>
    /// Multiplies two numbers.
    /// </summary>
    public double Multiply(double a, double b) => a * b;

    /// <summary>
    /// Divides a by b.
    /// </summary>
    /// <exception cref="DivideByZeroException">Thrown when b is zero.</exception>
    public double Divide(double a, double b)
    {
        if (b == 0)
            throw new DivideByZeroException("Cannot divide by zero");
        return a / b;
    }

    /// <summary>
    /// Calculates factorial of n.
    /// </summary>
    /// <exception cref="ArgumentException">Thrown when n is negative.</exception>
    public long Factorial(int n)
    {
        if (n < 0)
            throw new ArgumentException("Factorial of negative number", nameof(n));
        if (n == 0 || n == 1)
            return 1;
        
        long result = 1;
        for (int i = 2; i <= n; i++)
            result *= i;
        return result;
    }

    /// <summary>
    /// Calculates the nth Fibonacci number.
    /// </summary>
    /// <exception cref="ArgumentException">Thrown when n is negative.</exception>
    public long Fibonacci(int n)
    {
        if (n < 0)
            throw new ArgumentException("Fibonacci of negative number", nameof(n));
        if (n <= 1)
            return n;
        
        long a = 0, b = 1;
        for (int i = 2; i <= n; i++)
        {
            long temp = a + b;
            a = b;
            b = temp;
        }
        return b;
    }

    /// <summary>
    /// Checks if n is prime.
    /// </summary>
    public bool IsPrime(int n)
    {
        if (n < 2)
            return false;
        for (int i = 2; i <= Math.Sqrt(n); i++)
            if (n % i == 0)
                return false;
        return true;
    }

    /// <summary>
    /// Returns a raised to the power of b.
    /// </summary>
    public double Power(double a, int b) => Math.Pow(a, b);

    /// <summary>
    /// Returns the absolute value of a.
    /// </summary>
    public double Abs(double a) => Math.Abs(a);
}
