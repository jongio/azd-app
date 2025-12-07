using NUnit.Framework;
using NunitApi;

namespace NunitApi.Tests;

[TestFixture]
public class StringOperationsTests
{
    private StringOperations _stringOps = null!;

    [SetUp]
    public void Setup()
    {
        _stringOps = new StringOperations();
    }

    [Test]
    public void Capitalize_ReturnsCapitalizedString()
    {
        Assert.That(_stringOps.Capitalize("hello"), Is.EqualTo("Hello"));
    }

    [Test]
    public void Capitalize_EmptyString_ReturnsEmpty()
    {
        Assert.That(_stringOps.Capitalize(""), Is.EqualTo(string.Empty));
    }

    [Test]
    public void Capitalize_AllCaps_ReturnsProperCase()
    {
        Assert.That(_stringOps.Capitalize("HELLO"), Is.EqualTo("Hello"));
    }

    [Test]
    public void Reverse_ReturnsReversedString()
    {
        Assert.That(_stringOps.Reverse("hello"), Is.EqualTo("olleh"));
    }

    [Test]
    public void Reverse_EmptyString_ReturnsEmpty()
    {
        Assert.That(_stringOps.Reverse(""), Is.EqualTo(string.Empty));
    }

    [Test]
    public void Truncate_LongString_TruncatesWithSuffix()
    {
        Assert.That(_stringOps.Truncate("hello world", 8), Is.EqualTo("hello..."));
    }

    [Test]
    public void Truncate_ShortString_ReturnsOriginal()
    {
        Assert.That(_stringOps.Truncate("hello", 10), Is.EqualTo("hello"));
    }

    [Test]
    public void CountWords_Sentence_ReturnsWordCount()
    {
        Assert.That(_stringOps.CountWords("hello world"), Is.EqualTo(2));
    }

    [Test]
    public void CountWords_Empty_ReturnsZero()
    {
        Assert.That(_stringOps.CountWords(""), Is.EqualTo(0));
    }

    [Test]
    public void IsPalindrome_Palindrome_ReturnsTrue()
    {
        Assert.That(_stringOps.IsPalindrome("racecar"), Is.True);
    }

    [Test]
    public void IsPalindrome_SentenceWithSpaces_ReturnsTrue()
    {
        Assert.That(_stringOps.IsPalindrome("A man a plan a canal Panama"), Is.True);
    }

    [Test]
    public void IsPalindrome_NonPalindrome_ReturnsFalse()
    {
        Assert.That(_stringOps.IsPalindrome("hello"), Is.False);
    }

    [Test]
    public void CamelToSnake_CamelCase_ReturnsSnakeCase()
    {
        Assert.That(_stringOps.CamelToSnake("camelCase"), Is.EqualTo("camel_case"));
    }

    [Test]
    public void SnakeToCamel_SnakeCase_ReturnsCamelCase()
    {
        Assert.That(_stringOps.SnakeToCamel("snake_case"), Is.EqualTo("snakeCase"));
    }

    [Test]
    public void Slugify_TextWithSpaces_ReturnsSlugged()
    {
        Assert.That(_stringOps.Slugify("Hello World!"), Is.EqualTo("hello-world"));
    }

    [TestCase("banana", "an", 2)]
    [TestCase("hello", "x", 0)]
    [TestCase("aaa", "a", 3)]
    public void CountOccurrences_ReturnsCorrectCount(string text, string sub, int expected)
    {
        Assert.That(_stringOps.CountOccurrences(text, sub), Is.EqualTo(expected));
    }

    [Test]
    public void PadStart_AddsLeadingChars()
    {
        Assert.That(_stringOps.PadStart("5", 3, '0'), Is.EqualTo("005"));
    }

    [Test]
    public void PadEnd_AddsTrailingChars()
    {
        Assert.That(_stringOps.PadEnd("5", 3, '0'), Is.EqualTo("500"));
    }

    [Test]
    public void RemoveWhitespace_RemovesAllWhitespace()
    {
        Assert.That(_stringOps.RemoveWhitespace("hello world"), Is.EqualTo("helloworld"));
    }

    [Test]
    public void IsEmail_ValidEmail_ReturnsTrue()
    {
        Assert.That(_stringOps.IsEmail("test@example.com"), Is.True);
    }

    [Test]
    public void IsEmail_InvalidEmail_ReturnsFalse()
    {
        Assert.That(_stringOps.IsEmail("invalid"), Is.False);
    }

    [Test]
    public void ExtractNumbers_TextWithNumbers_ReturnsNumbers()
    {
        Assert.That(_stringOps.ExtractNumbers("abc123def456"), Is.EqualTo(new[] { 123, 456 }));
    }

    [Test]
    public void ExtractNumbers_NoNumbers_ReturnsEmpty()
    {
        Assert.That(_stringOps.ExtractNumbers("no numbers"), Is.Empty);
    }
}
