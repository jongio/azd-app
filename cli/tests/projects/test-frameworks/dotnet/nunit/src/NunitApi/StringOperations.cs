namespace NunitApi;

/// <summary>
/// String operations class demonstrating various string manipulation methods.
/// </summary>
public class StringOperations
{
    public string Capitalize(string text)
    {
        if (string.IsNullOrEmpty(text)) return string.Empty;
        return char.ToUpper(text[0]) + text[1..].ToLower();
    }
    
    public string Reverse(string text) => new(text.Reverse().ToArray());
    
    public string Truncate(string text, int maxLength, string suffix = "...")
    {
        if (text.Length <= maxLength) return text;
        return text[..(maxLength - suffix.Length)] + suffix;
    }
    
    public int CountWords(string text)
    {
        if (string.IsNullOrWhiteSpace(text)) return 0;
        return text.Split(new[] { ' ' }, StringSplitOptions.RemoveEmptyEntries).Length;
    }
    
    public bool IsPalindrome(string text)
    {
        var cleaned = new string(text.ToLower().Where(char.IsLetterOrDigit).ToArray());
        return cleaned.SequenceEqual(cleaned.Reverse());
    }
    
    public string CamelToSnake(string text)
    {
        return string.Concat(text.Select((c, i) =>
            i > 0 && char.IsUpper(c) ? "_" + char.ToLower(c) : char.ToLower(c).ToString()));
    }
    
    public string SnakeToCamel(string text)
    {
        var parts = text.Split('_');
        return parts[0] + string.Concat(parts.Skip(1).Select(p => 
            char.ToUpper(p[0]) + p[1..]));
    }
    
    public string Slugify(string text)
    {
        var slug = text.ToLower().Trim();
        slug = System.Text.RegularExpressions.Regex.Replace(slug, @"[^\w\s-]", "");
        slug = System.Text.RegularExpressions.Regex.Replace(slug, @"[\s_-]+", "-");
        return slug.Trim('-');
    }
    
    public int CountOccurrences(string text, string substring)
    {
        if (string.IsNullOrEmpty(substring)) return 0;
        
        int count = 0, pos = 0;
        while ((pos = text.IndexOf(substring, pos, StringComparison.Ordinal)) != -1)
        {
            count++;
            pos += substring.Length;
        }
        return count;
    }
    
    public string PadStart(string text, int length, char padChar = ' ')
    {
        return text.PadLeft(length, padChar);
    }
    
    public string PadEnd(string text, int length, char padChar = ' ')
    {
        return text.PadRight(length, padChar);
    }
    
    public string RemoveWhitespace(string text)
    {
        return System.Text.RegularExpressions.Regex.Replace(text, @"\s+", "");
    }
    
    public bool IsEmail(string text)
    {
        var pattern = @"^[^\s@]+@[^\s@]+\.[^\s@]+$";
        return System.Text.RegularExpressions.Regex.IsMatch(text, pattern);
    }
    
    public int[] ExtractNumbers(string text)
    {
        var matches = System.Text.RegularExpressions.Regex.Matches(text, @"\d+");
        return matches.Select(m => int.Parse(m.Value)).ToArray();
    }
}
