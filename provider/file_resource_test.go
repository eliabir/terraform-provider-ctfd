package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_File_Lifecycle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
resource "ctfd_challenge_standard" "example" {
	name        = "Example challenge"
	category    = "test"
	description = "Example challenge description..."
	value       = 500
}

resource "ctfd_file" "pouet" {
	challenge_id      = ctfd_challenge_standard.example.id
	name              = "pouet.txt"
	contentb64        = "UG91ZXQgaXMgYSBjbG93biBjYXQK"
	contentb64_hash   = base64sha256(base64decode("UG91ZXQgaXMgYSBjbG93biBjYXQK"))
}

resource "ctfd_file" "pouet_2" {
	name            = "pouet-2.txt"
	contentb64      = "UG91ZXQgaXMgYSBjbG93biBjYXQsIGJ1dCBoYXMgbm90IGNoYWxsZW5nZQo="
	contentb64_hash = base64sha256(base64decode("UG91ZXQgaXMgYSBjbG93biBjYXQsIGJ1dCBoYXMgbm90IGNoYWxsZW5nZQo="))
}
`,
			},
			// ImportState testing
			{
				ResourceName:            "ctfd_file.pouet",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"contentb64_hash"}, // contentb64_hash cant be reconstructed when importing resource
			}, {
				ResourceName:            "ctfd_file.pouet_2",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"contentb64_hash"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
resource "ctfd_challenge_standard" "example" {
	name        = "Example challenge"
	category    = "test"
	description = "Example challenge description..."
	value       = 500
}

resource "ctfd_file" "pouet" {
	challenge_id    = ctfd_challenge_standard.example.id
	name            = "pouet.txt"
	contentb64      = "UG91ZXQgdGhlIDJuZCBpcyB0aGUgY2xvd25pZXN0IGNhdCBldmVyCg=="
	contentb64_hash = base64sha256(base64decode("UG91ZXQgdGhlIDJuZCBpcyB0aGUgY2xvd25pZXN0IGNhdCBldmVyCg=="))
}

resource "ctfd_file" "pouet_2" {
	name            = "pouet-second.txt"
	contentb64      = "UG91ZXQgaXMgYSBjbG93biBjYXQsIGJ1dCBoYXMgbm90IGNoYWxsZW5nZQo="
	contentb64_hash = base64sha256(base64decode("UG91ZXQgaXMgYSBjbG93biBjYXQsIGJ1dCBoYXMgbm90IGNoYWxsZW5nZQo="))
}
`,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
