package delegated

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

func TestCoreMemberNumericStatus(t *testing.T) {
	for _, status := range []int{1, 2} {
		for _, operation := range []string{"page", "list", "get", "batch-get"} {
			t.Run(fmt.Sprintf("%s/%d", operation, status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer sts-token" {
						t.Error("credential mismatch")
					}
					item := fmt.Sprintf(`{"member_uuid":"member-1","tenant_uuid":"tenant-1","user_uuid":"user-1","display_name":"Example","status":%d}`, status)
					switch operation {
					case "get":
						fmt.Fprintf(w, `{"data":%s}`, item)
					case "batch-get":
						fmt.Fprintf(w, `{"data":{"items":[%s]}}`, item)
					default:
						fmt.Fprintf(w, `{"data":{"items":[%s],"pagination":{"page":1,"page_size":200,"total":1}}}`, item)
					}
				}))
				defer server.Close()
				client, err := NewCoreClient(CoreClientConfig{BaseURL: server.URL, Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "sts-token", nil })})
				if err != nil {
					t.Fatal(err)
				}
				var items []contracts.Member
				switch operation {
				case "get":
					var member *contracts.Member
					member, err = client.GetMember(context.Background(), "tenant-1", "member-1")
					if member != nil {
						items = []contracts.Member{*member}
					}
				case "batch-get":
					items, err = client.BatchGetMembers(context.Background(), "tenant-1", []string{"member-1"})
				case "page":
					var page *contracts.MemberPage
					page, err = client.ListMembersPage(context.Background(), "tenant-1", contracts.MemberPageRequest{Page: 1, PageSize: 200})
					if page != nil {
						items = page.Items
					}
				case "list":
					items, err = client.ListMembers(context.Background(), "tenant-1")
				}
				if err != nil {
					t.Fatal(err)
				}
				if len(items) != 1 || items[0].Status != fmt.Sprint(status) || items[0].MemberUUID != "member-1" || items[0].UserUUID != "user-1" {
					t.Fatalf("members=%#v", items)
				}
			})
		}
	}
}

func TestCoreMemberRejectsStringStatusAndCrossTenant(t *testing.T) {
	for _, item := range []string{
		`{"member_uuid":"member-1","tenant_uuid":"tenant-1","status":"1"}`,
		`{"member_uuid":"member-1","tenant_uuid":"tenant-2","status":1}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, `{"data":%s}`, item) }))
		client, err := NewCoreClient(CoreClientConfig{BaseURL: server.URL, Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "sts-token", nil })})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.GetMember(context.Background(), "tenant-1", "member-1")
		server.Close()
		if err == nil {
			t.Fatal("invalid response accepted")
		}
		if iamerrors.CodeOf(err) != iamerrors.CodeUpstreamDependency && iamerrors.CodeOf(err) != iamerrors.CodeMemberNotFound {
			t.Fatalf("error=%v", err)
		}
	}
}

func TestCoreBatchResolveUsesLeanResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tenant/iam/members:batch-resolve" {
			t.Errorf("path=%s", r.URL.Path)
		}
		fmt.Fprint(w, `{"data":{"items":[{"member_uuid":"member-1","user_uuid":"user-1","display_name":"Example"}],"missing_member_uuids":["member-2"]}}`)
	}))
	defer server.Close()
	client, err := NewCoreClient(CoreClientConfig{BaseURL: server.URL, Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "sts-token", nil })})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.BatchResolveMembers(context.Background(), "tenant-1", []string{"member-1", "member-2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].TenantUUID != "tenant-1" || result.Items[0].Status != "" || len(result.MissingMemberUUIDs) != 1 || result.MissingMemberUUIDs[0] != "member-2" {
		t.Fatalf("result=%#v", result)
	}
}
