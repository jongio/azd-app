namespace FailingTests;

public class FailingTestClass
{
    [Xunit.Fact]
    public void TestShouldFailAssertion()
    {
        // This test should fail
        var result = 1 + 1;
        var expected = 3;
        Xunit.Assert.Equal(expected, result);
    }

    [Xunit.Fact]
    public void TestShouldPass()
    {
        // This test should pass
        var result = 2 + 2;
        var expected = 4;
        Xunit.Assert.Equal(expected, result);
    }

    [Xunit.Fact]
    public void TestShouldFailString()
    {
        // This test should fail
        var result = "hello";
        var expected = "world";
        Xunit.Assert.Equal(expected, result);
    }
}
