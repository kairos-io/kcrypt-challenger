package client_test

import (
	"fmt"
	"os"
	"path"

	client "github.com/kairos-io/kairos-challenger/cmd/discovery/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	Describe("GetConfiguration", func() {
		var expectedServer string

		When("kcrypt section is defined in config", func() {
			var tempDir, filePath string

			BeforeEach(func() {
				expectedServer = "myserver.org"
				tempDir = GinkgoT().TempDir()
				filePath = path.Join(tempDir, "kcrypt-challenger.yaml")
				content := fmt.Sprintf(`
#cloud-init

# Irrelevant configuration, just to make sure it would be ignored
kairos:
  network_token: test

kcrypt:
  challenger_server: %s
`, expectedServer)
				os.WriteFile(
					filePath,
					[]byte(content),
					0744)
			})

			It("respects the config", func() {
				c, err := client.GetConfiguration(tempDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(c.Kcrypt.Server).To(Equal(expectedServer))
			})
		})
	})
})
