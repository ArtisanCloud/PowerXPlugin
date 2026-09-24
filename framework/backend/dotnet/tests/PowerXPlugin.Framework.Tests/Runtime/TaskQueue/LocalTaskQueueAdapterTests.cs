using PowerXPlugin.Framework.Runtime.TaskQueue;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.TaskQueue;
public sealed class LocalTaskQueueAdapterTests
{
    [Fact]
    public async Task Local_adapter_uses_injected_store_and_keeps_tenant_subscriber_isolation()
    {
        var queue = new LocalTaskQueueAdapter(new InMemoryTaskQueue());
        await queue.EnqueueAsync(new TaskQueueMessage("a", "tenant-a", "sub-a", "topic", []));
        await queue.EnqueueAsync(new TaskQueueMessage("b", "tenant-b", "sub-a", "topic", []));
        var a = await queue.DequeueAsync(new TaskQueueDequeueRequest("tenant-a", "sub-a"));
        var b = await queue.DequeueAsync(new TaskQueueDequeueRequest("tenant-b", "sub-a"));
        Assert.Equal("a", Assert.Single(a).Id);
        Assert.Equal("b", Assert.Single(b).Id);
    }
}
