namespace XunitApi;

/// <summary>
/// Calculator class demonstrating various mathematical operations.
/// </summary>
public class Calculator
{
    public int Add(int a, int b) => a + b;
    
    public int Subtract(int a, int b) => a - b;
    
    public int Multiply(int a, int b) => a * b;
    
    public double Divide(double a, double b)
    {
        if (b == 0)
            throw new DivideByZeroException("Cannot divide by zero");
        return a / b;
    }
    
    public double Power(double baseNum, double exponent) => Math.Pow(baseNum, exponent);
    
    public double Sqrt(double n)
    {
        if (n < 0)
            throw new ArgumentException("Cannot calculate square root of negative number");
        return Math.Sqrt(n);
    }
    
    public int Modulo(int a, int b)
    {
        if (b == 0)
            throw new DivideByZeroException("Cannot calculate modulo by zero");
        return a % b;
    }
    
    public int Factorial(int n)
    {
        if (n < 0)
            throw new ArgumentException("Cannot calculate factorial of negative number");
        if (n <= 1) return 1;
        
        int result = 1;
        for (int i = 2; i <= n; i++)
            result *= i;
        return result;
    }
    
    public int Fibonacci(int n)
    {
        if (n < 0)
            throw new ArgumentException("Cannot calculate fibonacci of negative number");
        if (n <= 1) return n;
        
        int a = 0, b = 1;
        for (int i = 2; i <= n; i++)
        {
            int temp = a + b;
            a = b;
            b = temp;
        }
        return b;
    }
    
    public bool IsPrime(int n)
    {
        if (n < 2) return false;
        if (n == 2) return true;
        if (n % 2 == 0) return false;
        
        for (int i = 3; i <= Math.Sqrt(n); i += 2)
        {
            if (n % i == 0) return false;
        }
        return true;
    }
    
    public int Gcd(int a, int b)
    {
        a = Math.Abs(a);
        b = Math.Abs(b);
        while (b != 0)
        {
            int temp = b;
            b = a % b;
            a = temp;
        }
        return a;
    }
    
    public int Lcm(int a, int b) => Math.Abs(a * b) / Gcd(a, b);
}
