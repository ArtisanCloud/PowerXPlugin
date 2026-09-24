using System.Text.Json;
using PowerXPlugin.Framework.Runtime.TaskCenter;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.TaskCenter;
public sealed class TaskCenterTransitionTests
{
    private const string Tenant="00000000-0000-4000-8000-000000000001", Task="00000000-0000-4000-8000-000000000002";
    [Fact] public void Allows_monotonic_running_transition() => TaskCenterTransitions.Validate(Current(), new TaskScope(Tenant), Update(TaskState.Running, 30));
    [Fact] public void Rejects_revision_regression_and_terminal_overwrite() { var cas=Assert.Throws<TaskCenterException>(()=>TaskCenterTransitions.Validate(Current(),new TaskScope(Tenant),Update(TaskState.Running,30,2))); Assert.Equal(TaskCenterErrors.Conflict,cas.Code); var done=Current(TaskState.Succeeded,100,completed:true); var terminal=Assert.Throws<TaskCenterException>(()=>TaskCenterTransitions.Validate(done,new TaskScope(Tenant),Update(TaskState.Failed,100))); Assert.Equal(TaskCenterErrors.Conflict,terminal.Code); }
    [Fact] public void Rejects_progress_regression_and_invalid_success() { Assert.Equal(TaskCenterErrors.Conflict,Assert.Throws<TaskCenterException>(()=>TaskCenterTransitions.Validate(Current(),new TaskScope(Tenant),Update(TaskState.Running,10))).Code); Assert.Equal(TaskCenterErrors.InvalidArgument,Assert.Throws<TaskCenterException>(()=>TaskCenterTransitions.Validate(Current(),new TaskScope(Tenant),Update(TaskState.Succeeded,50))).Code); }
    private static TaskRecord Current(TaskState state=TaskState.Running,int progress=20,bool completed=false) => new(Task,Tenant,"export",state,progress,1,null,JsonDocument.Parse("null").RootElement.Clone(),DateTimeOffset.UtcNow,DateTimeOffset.UtcNow,completed?DateTimeOffset.UtcNow:null);
    private static TaskUpdateRequest Update(TaskState state,int progress,long revision=1)=>new(Task,revision,state,progress,null,JsonDocument.Parse("null").RootElement.Clone());
}
