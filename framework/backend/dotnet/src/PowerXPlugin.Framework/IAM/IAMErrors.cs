namespace PowerXPlugin.Framework.IAM;

public static class IAMErrors
{
    public const string CodeInvalidMode = "IAM_MODE_INVALID";
    public const string CodeModeConflict = "IAM_MODE_CONFLICT";
    public const string CodeAdapterNotBound = "IAM_ADAPTER_NOT_BOUND";
    public const string CodeAdapterAlreadyBound = "IAM_ADAPTER_ALREADY_BOUND";
    public const string CodeUnauthorized = "IAM_UNAUTHORIZED";
    public const string CodeForbidden = "IAM_FORBIDDEN";
    public const string CodeUpstreamDependency = "IAM_UPSTREAM_DEPENDENCY";

    public static int StatusCode(string code) => code switch
    {
        CodeInvalidMode => 400,
        CodeModeConflict => 409,
        CodeAdapterNotBound => 424,
        CodeAdapterAlreadyBound => 409,
        CodeUnauthorized => 401,
        CodeForbidden => 403,
        CodeUpstreamDependency => 424,
        _ => 500
    };
}
