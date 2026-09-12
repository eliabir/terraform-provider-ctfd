package provider_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_File_Lifecycle(t *testing.T) {
	// Absolute paths so both Terraform's filesha256 (evaluated in its temporary
	// working directory) and the provider's os.ReadFile (relative to the test
	// process working directory) resolve to the same fixture files.
	pouet := abspath(t, "testdata/pouet.txt")
	pouet2 := abspath(t, "testdata/pouet-2.txt")
	pouetUpdated := abspath(t, "testdata/pouet-updated.txt")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "ctfd_challenge_standard" "example" {
	name        = "Example challenge"
	category    = "test"
	description = "Example challenge description..."
	value       = 500
}

resource "ctfd_file" "pouet" {
	challenge_id = ctfd_challenge_standard.example.id
	name         = "pouet.txt"
	content_path = %[1]q
	content_hash = filesha256(%[1]q)
}

resource "ctfd_file" "pouet_2" {
	name         = "pouet-2.txt"
	content_path = %[2]q
	content_hash = filesha256(%[2]q)
}
`, pouet, pouet2),
			},
			// ImportState testing
			{
				ResourceName:            "ctfd_file.pouet",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"content_path", "content_hash"}, // neither can be reconstructed when importing resource
			}, {
				ResourceName:            "ctfd_file.pouet_2",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"content_path", "content_hash"},
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "ctfd_challenge_standard" "example" {
	name        = "Example challenge"
	category    = "test"
	description = "Example challenge description..."
	value       = 500
}

resource "ctfd_file" "pouet" {
	challenge_id = ctfd_challenge_standard.example.id
	name         = "pouet.txt"
	content_path = %[1]q
	content_hash = filesha256(%[1]q)
}

resource "ctfd_file" "pouet_2" {
	name         = "pouet-second.txt"
	content_path = %[2]q
	content_hash = filesha256(%[2]q)
}
`, pouetUpdated, pouet2),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func abspath(t *testing.T, rel string) string {
	t.Helper()
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatalf("could not resolve %q: %s", rel, err)
	}
	return abs
}
