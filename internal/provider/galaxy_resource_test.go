package provider

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var galaxyPath string

func TestAccGalaxyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccGalaxyResourceConfig("v1.2.5"),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						return testAccGalaxyResourceAssertConfig(s, "my.role", "v1.2.5")
					},
				),
			},
			// Ensure idempotency
			{
				Config: testAccGalaxyResourceConfig("v1.2.5"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			// Ensure the resources are recreated if they're missing
			{
				PreConfig: func() {
					if err := os.RemoveAll(galaxyPath); err != nil {
						t.Fatal("remove the galaxy resource" + err.Error())
					}
				},
				Config: testAccGalaxyResourceConfig("v1.2.5"),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						return testAccGalaxyResourceAssertConfig(s, "my.role", "v1.2.5")
					},
				),
			},
			// Update and Read Testing
			{
				Config: testAccGalaxyResourceConfig("v1.2.6"),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						return testAccGalaxyResourceAssertConfig(s, "my.role", "v1.2.6")
					},
				),
			},
		},
		CheckDestroy: func(s *terraform.State) error {
			host := s.RootModule().Resources["ansible_galaxy.test"].Primary
			path := host.Attributes["path"]

			_, err := os.Stat(path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return nil
				}

				return fmt.Errorf("unexpected errror: %w", err)
			}

			return errors.New("the file still exists")
		},
	})
}

func testAccGalaxyResourceConfig(version string) string {
	role := "singleplatform-eng.users"
	return fmt.Sprintf(`
	resource "ansible_galaxy" "test" {
		role = "%s"
		version = "%s"
		name = "my.role"
	}
	`, role, version)
}

func testAccGalaxyResourceAssertConfig(s *terraform.State, name, version string) error {
	host := s.RootModule().Resources["ansible_galaxy.test"].Primary
	path := host.Attributes["path"]

	galaxyPath = path

	out, err := exec.Command("ansible-galaxy", "list", "--roles-path", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(out))
	}

	if !strings.Contains(string(out), fmt.Sprintf("%s, %s", name, version)) {
		return errors.New("missing role")
	}

	return nil
}
