namespace PowerXPlugin.Framework.IAM;

public static class IAMErrors
{
    public const string CodeInvalidMode = "IAM_ADAPTER_MODE_INVALID";
    public const string CodeModeConflict = "IAM_ADAPTER_MODE_CONFLICT";
    public const string CodeAdapterNotBound = "IAM_ADAPTER_NOT_BOUND";
    public const string CodeAdapterAlreadyBound = "IAM_ADAPTER_ALREADY_BOUND";
    public const string CodeUnauthorized = "IAM_UNAUTHORIZED";
    public const string CodeForbidden = "IAM_FORBIDDEN";
    public const string CodeUpstreamDependency = "IAM_UPSTREAM_DEPENDENCY";
    public const string CodeIdentityMigrationRequired = "IAM_IDENTITY_MIGRATION_REQUIRED";

    public static int StatusCode(string code) => code switch
    {
        "IAM_CONTEXT_WS_HANDLE_REQUIRED" or "IAM_CONTEXT_WS_HANDLE_INVALID" or "IAM_CONTEXT_WS_HANDLE_EXPIRED"=>401,
        "IAM_WS_IDENTITY_MISMATCH" or "IAM_WS_TOPIC_FORBIDDEN"=>403,
        "IAM_CONTEXT_PROOF_REQUIRED" or "IAM_CONTEXT_PROOF_INVALID" or "IAM_CONTEXT_PROOF_EXPIRED"
            or "IAM_CONTEXT_USER_SESSION_INVALID" or "IAM_CONTEXT_USER_SESSION_EXPIRED"
            or "IAM_CONTEXT_SERVICE_CREDENTIAL_INVALID" or "IAM_CONTEXT_SERVICE_CREDENTIAL_EXPIRED" => 401,
        "IAM_CONTEXT_SERVICE_FORBIDDEN" or "IAM_CONTEXT_USER_FORBIDDEN" or "IAM_CONTEXT_MEMBER_REQUIRED"
            or "IAM_CONTEXT_REQUEST_FORBIDDEN" or "IAM_CONTEXT_TENANT_FORBIDDEN" or "IAM_CONTEXT_PLUGIN_FORBIDDEN" => 403,
        "IAM_CONTEXT_POLICY_CHANGED" or "IAM_IDENTITY_MIGRATION_REQUIRED" => 409,
        "IAM_CONTEXT_INVALID_ARGUMENT" => 400,
        "IAM_CONTEXT_UNAVAILABLE" => 503,
        "IAM_CONTEXT_RESPONSE_INVALID" => 502,
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
