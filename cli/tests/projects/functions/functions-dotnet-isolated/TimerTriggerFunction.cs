using Microsoft.Azure.Functions.Worker;
using Microsoft.Extensions.Logging;

namespace FunctionApp;

public class TimerTriggerFunction
{
    private readonly ILogger _logger;

    public TimerTriggerFunction(ILoggerFactory loggerFactory)
    {
        _logger = loggerFactory.CreateLogger<TimerTriggerFunction>();
    }

    [Function("TimerTrigger")]
    public void Run([TimerTrigger("0 */5 * * * *")] TimerInfo myTimer)
    {
        var timestamp = DateTime.UtcNow.ToString("o");
        _logger.LogInformation($"C# Timer trigger function executed at: {timestamp}");

        if (myTimer.ScheduleStatus is not null)
        {
            _logger.LogInformation($"Next scheduled run: {myTimer.ScheduleStatus.Next}");
        }
    }
}

public class TimerInfo
{
    public TimerScheduleStatus? ScheduleStatus { get; set; }
}

public class TimerScheduleStatus
{
    public DateTime Last { get; set; }
    public DateTime Next { get; set; }
    public DateTime LastUpdated { get; set; }
}
