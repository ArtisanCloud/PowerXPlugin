using System.Text;
using System.Globalization;
using System.Text.Json;
using System.Text.Json.Nodes;
using System.Text.Json.Serialization;
using System.Text.RegularExpressions;
using PowerXPlugin.Framework.Runtime.Capability;

namespace PowerXPlugin.Framework.Runtime.Customer;

public sealed record CustomerManagementScope(string TenantUuid, string TraceId = "");
public sealed record CustomerList(int Page = 1, int PageSize = 20, string Q = "", string Status = "");
public sealed record CustomerPage<T>(IReadOnlyList<T> Items, long Total, int Page, int PageSize);
public sealed record PrimaryContactInput(string DisplayName = "", string GivenName = "", string FamilyName = "", string Email = "", string Phone = "");
public sealed record BasicCustomerInput(string Type = "person", PrimaryContactInput? PrimaryContact = null, string Status = "active", string PrimaryEmail = "", string PrimaryPhone = "", string DisplayName = "", string Nickname = "", string GivenName = "", string FamilyName = "", string AvatarUrl = "", string Locale = "zh-CN", string Timezone = "Asia/Shanghai");
public sealed record CustomerProfilePatch(string? DisplayName = null, string? Nickname = null, string? GivenName = null, string? FamilyName = null, string? PrimaryEmail = null, string? PrimaryPhone = null, string? AvatarUrl = null, string? Locale = null, string? Timezone = null, string? Status = null);
public sealed record ManagedCustomer(string CustomerUuid, string Type, string PrimaryContactUuid, string Status, string PrimaryEmail, string PrimaryPhone, string DisplayName, string Nickname, string GivenName, string FamilyName, string AvatarUrl, string Locale, string Timezone, DateTime CreatedAt, DateTime UpdatedAt);
public sealed record ContactInput(string DisplayName, string Status = "active", string CreationIntent = "explicit_create", string GivenName = "", string FamilyName = "", string Email = "", string Phone = "", IReadOnlyList<string>? Roles = null, IReadOnlyList<string>? Tags = null);
public sealed record ContactPatch(string? DisplayName = null, string? GivenName = null, string? FamilyName = null, string? Email = null, string? Phone = null, string? Status = null, IReadOnlyList<string>? Roles = null, IReadOnlyList<string>? Tags = null);
public sealed record ManagedContact(string ContactUuid, string TenantUuid, string CustomerUuid, string DisplayName, string GivenName, string FamilyName, string Email, string Phone, string Status, IReadOnlyList<string> Roles, IReadOnlyList<string> Tags, DateTime CreatedAt, DateTime UpdatedAt);
public sealed record ContactIdentityInput(string ChannelDictionaryItemUuid, string ExternalSubject);
public sealed record ManagedContactIdentity(string IdentityUuid, string TenantUuid, string CustomerUuid, string ContactUuid, string ChannelDictionaryItemUuid, string ExternalSubject, string Status);
public sealed record ContactIdentityMatch(ManagedContact Contact, ManagedContactIdentity Identity);
public sealed record ManagedExternalIdentity(string IdentityUuid, string CustomerUuid, string ProviderSubject, string Status, string Type, string PrimaryContactUuid);
public sealed record ExternalIdentityLookup(bool Found, ManagedExternalIdentity? Item = null);

public interface ICustomerManagementService
{
    Task<CustomerPage<ManagedCustomer>> ListAsync(CustomerManagementScope scope, CustomerList query, CancellationToken ct = default);
    Task<ManagedCustomer> CreateAsync(CustomerManagementScope scope, BasicCustomerInput input, CancellationToken ct = default);
    Task<ManagedCustomer> UpdateAsync(CustomerManagementScope scope, string customerUuid, CustomerProfilePatch patch, CancellationToken ct = default);
}
public interface IContactService
{
    Task<CustomerPage<ManagedContact>> ListAsync(CustomerManagementScope scope, string customerUuid, CustomerList query, CancellationToken ct = default);
    Task<ManagedContact> GetAsync(CustomerManagementScope scope, string customerUuid, string contactUuid, CancellationToken ct = default);
    Task<ManagedContact> CreateAsync(CustomerManagementScope scope, string customerUuid, ContactInput input, CancellationToken ct = default);
    Task<ManagedContact> UpdateAsync(CustomerManagementScope scope, string customerUuid, string contactUuid, ContactPatch patch, CancellationToken ct = default);
    Task<ContactIdentityMatch> ResolveIdentityAsync(CustomerManagementScope scope, string customerUuid, ContactIdentityInput input, CancellationToken ct = default);
    Task<ManagedContactIdentity> BindIdentityAsync(CustomerManagementScope scope, string customerUuid, string contactUuid, ContactIdentityInput input, CancellationToken ct = default);
}
public interface ICustomerExternalIdentityManagement
{
    Task<ExternalIdentityLookup> LookupAsync(CustomerManagementScope scope, string subject, CancellationToken ct = default);
    Task<CustomerPage<ManagedExternalIdentity>> ListAsync(CustomerManagementScope scope, string customerUuid, CustomerList query, CancellationToken ct = default);
    Task<ManagedExternalIdentity> BindAsync(CustomerManagementScope scope, string customerUuid, string subject, CancellationToken ct = default);
    Task<ManagedExternalIdentity> CreateAndBindAsync(CustomerManagementScope scope, string subject, BasicCustomerInput customer, CancellationToken ct = default);
}

public sealed class CustomerManagementException(string code) : InvalidOperationException(code) { public string Code { get; } = code; }
public static class CustomerManagementValidation
{
    public static void Uuid(string value) { if (!Guid.TryParse(value, out var id) || id == Guid.Empty || id.ToString() != value) Fail("CUSTOMER_ACCOUNT_INVALID_ARGUMENT"); }
    public static void Scope(CustomerManagementScope scope) => Uuid(scope.TenantUuid);
    public static void Fail(string code) => throw new CustomerManagementException(code);
    public static void Customer(BasicCustomerInput input)
    {
        if (input.Type is not ("person" or "company")) Fail("CUSTOMER_TYPE_REQUIRED");
        if (input.Status is not ("active" or "pending" or "suspended" or "disabled")) Fail("CUSTOMER_ACCOUNT_STATUS_INVALID");
        if (input.Type == "company" && string.IsNullOrWhiteSpace(input.PrimaryContact?.DisplayName)) Fail("CUSTOMER_PRIMARY_CONTACT_REQUIRED");
        if (string.IsNullOrWhiteSpace(input.DisplayName) && string.IsNullOrWhiteSpace(input.Nickname) && string.IsNullOrWhiteSpace(input.PrimaryEmail) && string.IsNullOrWhiteSpace(input.GivenName + input.FamilyName)) Fail("CUSTOMER_ACCOUNT_IDENTITY_REQUIRED");
    }
    public static void Contact(string displayName, string status, IReadOnlyList<string>? roles, IReadOnlyList<string>? tags)
    {
        if (string.IsNullOrWhiteSpace(displayName) || displayName.Length > 128 || status is not ("active" or "inactive" or "temporary")) Fail("CONTACT_INVALID_ARGUMENT");
        if (roles?.Any(r => r is not ("primary" or "legal_representative")) == true || roles?.Distinct().Count() != roles?.Count) Fail("CONTACT_INVALID_ARGUMENT");
        if (tags is not null && (tags.Count > 20 || tags.Sum(t => Encoding.UTF8.GetByteCount(t)) > 1024 || tags.Any(t => !Regex.IsMatch(t, "^[a-z0-9][a-z0-9._-]{0,63}$")) || tags.Distinct().Count() != tags.Count)) Fail("CONTACT_INVALID_ARGUMENT");
    }
    public static void Intent(ContactInput input)
    {
        Contact(input.DisplayName, input.Status, input.Roles, input.Tags);
        if (input.CreationIntent != (input.Status == "temporary" ? "explicit_temporary" : "explicit_create")) Fail("CONTACT_INVALID_ARGUMENT");
    }
    public static void Subject(string subject) { if (string.IsNullOrWhiteSpace(subject) || Encoding.UTF8.GetByteCount(subject) > 255) Fail("CUSTOMER_ACCOUNT_INVALID_ARGUMENT"); }
}

/// <summary>Fixed Core service bindings. No endpoint, credential, tenant or raw forwarding input is exposed.</summary>
public sealed class PowerXCustomerManagementClient(ICapabilityRegistry registry, string tenantUuid) : ICustomerManagementService, IContactService, ICustomerExternalIdentityManagement
{
    private static readonly JsonSerializerOptions Json = new(JsonSerializerDefaults.Web) { PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower, DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull, Converters = { new CoreTimestampConverter() } };
    // Core account selectors return PostgreSQL timestamptz text; contact DTOs use RFC3339.
    private sealed class CoreTimestampConverter : JsonConverter<DateTime>
    {
        public override DateTime Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
        {
            if (reader.TokenType != JsonTokenType.String) throw new JsonException();
            if (reader.TryGetDateTime(out var iso)) return iso;
            var text = reader.GetString()!;
            if (!Regex.IsMatch(text, @"^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d{1,6})?[+-]\d{2}(:\d{2})?$")) throw new JsonException();
            if (Regex.IsMatch(text, @"[+-]\d{2}$")) text += ":00";
            if (!DateTimeOffset.TryParseExact(text, "yyyy-MM-dd HH:mm:ss.FFFFFFFzzz", CultureInfo.InvariantCulture, DateTimeStyles.None, out var timestamp)) throw new JsonException();
            return timestamp.UtcDateTime;
        }
        public override void Write(Utf8JsonWriter writer, DateTime value, JsonSerializerOptions options) => writer.WriteStringValue(value);
    }
    private async Task<JsonNode> Invoke(CustomerManagementScope scope, string aggregate, bool write, string operation, object? input, CancellationToken ct)
    {
        CustomerManagementValidation.Scope(scope);
        if (scope.TenantUuid != tenantUuid) throw new CustomerManagementException("CUSTOMER_TENANT_MISMATCH");
        var body = input is null ? new JsonObject() : JsonSerializer.SerializeToNode(input, Json)!.AsObject();
        body["operation"] = operation;
        var result = await registry.InvokeAsync(new($"com.corex.customer.{aggregate}.service_{(write ? "manage" : "read")}", null, null, "core_internal", null, scope.TraceId, null,
            new Dictionary<string, object?> { ["method"] = "INVOKE", ["endpoint"] = "core://customer/" + aggregate.Replace('_', '-'), ["body"] = body }, null), ct);
        if (result.FallbackUsed || result.ProtocolUsed != "core_internal" || result.Status != "completed") throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE");
        return JsonSerializer.SerializeToNode(result.Payload, Json) ?? throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE");
    }
    private static T Decode<T>(JsonNode node)
    {
        // Core Contact inherits its camelCase timestamp fields from BaseModel.
        if (typeof(T) == typeof(ManagedContact)) ContactTimestamps(node);
        else if (typeof(T) == typeof(CustomerPage<ManagedContact>)) foreach (var item in node["items"]!.AsArray()) ContactTimestamps(item!);
        else if (typeof(T) == typeof(ContactIdentityMatch)) ContactTimestamps(node["contact"]!);
        try { return node.Deserialize<T>(Json) ?? throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE"); }
        catch (JsonException) { throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE"); }
    }
    private static void ContactTimestamps(JsonNode node)
    {
        foreach (var (wire, field) in new[] { ("createdAt", "created_at"), ("updatedAt", "updated_at") })
            if (node[wire] is not null) node[field] = node[wire]!.DeepClone();
    }
    private static JsonNode Item(JsonNode node, string uuidField)
    {
        var item = node["item"] ?? throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE");
        if (item["uuid"] is not null) item[uuidField] = item["uuid"]!.DeepClone();
        CustomerManagementValidation.Uuid(item[uuidField]?.GetValue<string>() ?? "");
        return item;
    }
    private static CustomerPage<T> Page<T>(JsonNode node, string? uuidField = null)
    {
        if (node["items"] is not JsonArray items || node["page"]?.GetValue<int>() is not > 0 || node["page_size"]?.GetValue<int>() is not > 0 || node["total"] is null || node["total"]!.GetValue<long>() < items.Count) throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE");
        if (uuidField is not null) foreach (var item in items) { if (item?["uuid"] is not null) item[uuidField] = item["uuid"]!.DeepClone(); CustomerManagementValidation.Uuid(item?[uuidField]?.GetValue<string>() ?? ""); }
        return Decode<CustomerPage<T>>(node);
    }
    public async Task<CustomerPage<ManagedCustomer>> ListAsync(CustomerManagementScope s, CustomerList q, CancellationToken ct = default) => Page<ManagedCustomer>(await Invoke(s, "accounts", false, "list", q, ct), "customer_uuid");
    public async Task<ManagedCustomer> CreateAsync(CustomerManagementScope s, BasicCustomerInput input, CancellationToken ct = default) { CustomerManagementValidation.Customer(input); var item = Decode<ManagedCustomer>(Item(await Invoke(s, "accounts", true, "create", input, ct), "customer_uuid")); CustomerManagementValidation.Uuid(item.PrimaryContactUuid); return item; }
    public async Task<ManagedCustomer> UpdateAsync(CustomerManagementScope s, string id, CustomerProfilePatch input, CancellationToken ct = default) { CustomerManagementValidation.Uuid(id); var body = JsonSerializer.SerializeToNode(input, Json)!.AsObject(); body["customer_uuid"] = id; var item = Decode<ManagedCustomer>(Item(await Invoke(s, "accounts", true, "update", body, ct), "customer_uuid")); if (item.CustomerUuid != id) throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE"); CustomerManagementValidation.Uuid(item.PrimaryContactUuid); return item; }
    public async Task<CustomerPage<ManagedContact>> ListAsync(CustomerManagementScope s, string customer, CustomerList q, CancellationToken ct = default) { CustomerManagementValidation.Uuid(customer); var page = Page<ManagedContact>(await Invoke(s, "contacts", false, "list_by_customer", new { customer_uuid = customer, q.Page, q.PageSize, q.Q, q.Status }, ct), "contact_uuid"); if (page.Items.Any(x => x.CustomerUuid != customer || x.TenantUuid != s.TenantUuid)) throw new CustomerManagementException("CONTACT_CUSTOMER_MISMATCH"); return page; }
    public async Task<ManagedContact> GetAsync(CustomerManagementScope s, string customer, string id, CancellationToken ct = default) => await ContactItem(s, customer, id, false, "get", null, ct);
    public async Task<ManagedContact> CreateAsync(CustomerManagementScope s, string customer, ContactInput input, CancellationToken ct = default) { CustomerManagementValidation.Intent(input); return await ContactItem(s, customer, null, true, "create", input, ct); }
    public async Task<ManagedContact> UpdateAsync(CustomerManagementScope s, string customer, string id, ContactPatch input, CancellationToken ct = default) => await ContactItem(s, customer, id, true, "update", input, ct);
    private async Task<ManagedContact> ContactItem(CustomerManagementScope s, string customer, string? id, bool write, string operation, object? input, CancellationToken ct)
    {
        CustomerManagementValidation.Uuid(customer); if (id is not null) CustomerManagementValidation.Uuid(id);
        var body = input is null ? new JsonObject() : JsonSerializer.SerializeToNode(input, Json)!.AsObject(); body["customer_uuid"] = customer; if (id is not null) body["contact_uuid"] = id;
        var item = Decode<ManagedContact>(Item(await Invoke(s, "contacts", write, operation, body, ct), "contact_uuid"));
        if (item.CustomerUuid != customer || item.TenantUuid != s.TenantUuid || (id is not null && item.ContactUuid != id)) throw new CustomerManagementException("CONTACT_CUSTOMER_MISMATCH"); return item;
    }
    public async Task<ContactIdentityMatch> ResolveIdentityAsync(CustomerManagementScope s, string customer, ContactIdentityInput input, CancellationToken ct = default)
    {
        IdentityInput(customer, input);
        var item = (await Invoke(s, "contacts", false, "resolve_identity", new { customer_uuid = customer, input.ChannelDictionaryItemUuid, input.ExternalSubject }, ct))["item"] ?? throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE");
        var contact = item["contact"] ?? throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE");
        var identity = item["identity"] ?? throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE");
        contact["contact_uuid"] = contact["uuid"]?.DeepClone(); identity["identity_uuid"] = identity["uuid"]?.DeepClone();
        var result = Decode<ContactIdentityMatch>(item); CheckIdentity(s, customer, result.Contact.ContactUuid, input, result.Identity);
        if (result.Contact.TenantUuid != s.TenantUuid || result.Contact.CustomerUuid != customer) throw new CustomerManagementException("CONTACT_CUSTOMER_MISMATCH"); return result;
    }
    public async Task<ManagedContactIdentity> BindIdentityAsync(CustomerManagementScope s, string customer, string contact, ContactIdentityInput input, CancellationToken ct = default)
    {
        IdentityInput(customer, input); CustomerManagementValidation.Uuid(contact);
        var result = Decode<ManagedContactIdentity>(Item(await Invoke(s, "contacts", true, "bind_identity", new { customer_uuid = customer, contact_uuid = contact, input.ChannelDictionaryItemUuid, input.ExternalSubject }, ct), "identity_uuid"));
        CheckIdentity(s, customer, contact, input, result); return result;
    }
    private static void IdentityInput(string customer, ContactIdentityInput input) { CustomerManagementValidation.Uuid(customer); CustomerManagementValidation.Uuid(input.ChannelDictionaryItemUuid); CustomerManagementValidation.Subject(input.ExternalSubject); }
    private static void CheckIdentity(CustomerManagementScope s, string customer, string contact, ContactIdentityInput input, ManagedContactIdentity result)
    {
        CustomerManagementValidation.Uuid(result.IdentityUuid); CustomerManagementValidation.Uuid(contact);
        if (result.TenantUuid != s.TenantUuid || result.CustomerUuid != customer || result.ContactUuid != contact || result.ChannelDictionaryItemUuid != input.ChannelDictionaryItemUuid || result.ExternalSubject != input.ExternalSubject) throw new CustomerManagementException("CONTACT_CUSTOMER_MISMATCH");
    }
    public async Task<ExternalIdentityLookup> LookupAsync(CustomerManagementScope s, string subject, CancellationToken ct = default) { CustomerManagementValidation.Subject(subject); var node = await Invoke(s, "external_identities", false, "lookup", new { provider_subject = subject }, ct); if (node["found"] is null) throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE"); var result = Decode<ExternalIdentityLookup>(node); if (result.Found) { if (result.Item is null) throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE"); CheckExternal(result.Item, subject); } else if (result.Item is not null) throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE"); return result; }
    async Task<CustomerPage<ManagedExternalIdentity>> ICustomerExternalIdentityManagement.ListAsync(CustomerManagementScope s, string customer, CustomerList q, CancellationToken ct)
    {
        CustomerManagementValidation.Uuid(customer); var page = Page<ManagedExternalIdentity>(await Invoke(s, "external_identities", false, "list_by_customer", new { customer_uuid = customer, q.Page, q.PageSize }, ct), "identity_uuid");
        foreach (var item in page.Items) { CheckExternal(item, item.ProviderSubject); if (item.CustomerUuid != customer) throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE"); } return page;
    }
    public async Task<ManagedExternalIdentity> BindAsync(CustomerManagementScope s, string customer, string subject, CancellationToken ct = default) { CustomerManagementValidation.Subject(subject); CustomerManagementValidation.Uuid(customer); var item = Decode<ManagedExternalIdentity>(Item(await Invoke(s, "external_identities", true, "bind", new { customer_uuid = customer, provider_subject = subject }, ct), "identity_uuid")); CheckExternal(item, subject); if (item.CustomerUuid != customer) throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE"); return item; }
    public async Task<ManagedExternalIdentity> CreateAndBindAsync(CustomerManagementScope s, string subject, BasicCustomerInput customer, CancellationToken ct = default) { CustomerManagementValidation.Subject(subject); CustomerManagementValidation.Customer(customer); var item = Decode<ManagedExternalIdentity>(Item(await Invoke(s, "external_identities", true, "create_and_bind", new { provider_subject = subject, customer = new { customer.Type, customer.PrimaryContact, customer.DisplayName, customer.Nickname, customer.GivenName, customer.FamilyName, customer.PrimaryEmail, customer.PrimaryPhone } }, ct), "identity_uuid")); CheckExternal(item, subject); CustomerManagementValidation.Uuid(item.PrimaryContactUuid); if (item.Type != customer.Type) throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE"); return item; }
    private static void CheckExternal(ManagedExternalIdentity item, string subject) { CustomerManagementValidation.Uuid(item.IdentityUuid); CustomerManagementValidation.Uuid(item.CustomerUuid); if (item.ProviderSubject != subject || item.Status != "active") throw new CustomerManagementException("CUSTOMER_ACCOUNT_INVALID_RESPONSE"); }
}
