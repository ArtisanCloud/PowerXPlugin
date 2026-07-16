namespace PowerXPlugin.Framework.Routing;

public class ResponseEnvelope
{
    public bool Success { get; set; }
    public object? Data { get; set; }
    public ErrorInfo? Error { get; set; }
    public string? Message { get; set; }
    public DateTime Timestamp { get; set; } = DateTime.UtcNow;
    public string? RequestID { get; set; }

    public record ErrorInfo
    {
        public string Code { get; set; } = "";
        public string? Message { get; set; }
        public object? Details { get; set; }
    }
}

public static class ResponseHelper
{
    public static IResult Success(object? data = null, string? message = null, int status = 200)
    {
        return Results.Json(new ResponseEnvelope
        {
            Success = true,
            Data = data,
            Message = message
        }, statusCode: status);
    }

    public static IResult Error(string code, string? message = null, object? details = null, int status = 400)
    {
        return Results.Json(new ResponseEnvelope
        {
            Success = false,
            Error = new ResponseEnvelope.ErrorInfo
            {
                Code = code,
                Message = message,
                Details = details
            }
        }, statusCode: status);
    }
}
