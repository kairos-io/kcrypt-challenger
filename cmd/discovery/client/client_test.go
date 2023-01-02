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

		When("environment variable is set", func() {
			BeforeEach(func() {
				expectedServer = "myserver.org"
				GinkgoT().Setenv("WSS_SERVER", expectedServer)
			})
			It("respects the environment variable", func() {
				c, err := client.GetConfiguration("")
				Expect(err).ToNot(HaveOccurred())
				Expect(c.Server).To(Equal(expectedServer))
			})
		})

		When("kcrypt-challenger.conf is present", func() {
			var filePath string

			BeforeEach(func() {
				expectedServer = "myserver.org"
				tempDir := GinkgoT().TempDir()
				filePath = path.Join(tempDir, "kcrypt-challenger.conf")
				content := fmt.Sprintf(`
---
challenger_server: %s
`, expectedServer)
				os.WriteFile(
					filePath,
					[]byte(content),
					0744)
			})

			It("respects the config file", func() {
				c, err := client.GetConfiguration(filePath)
				Expect(err).ToNot(HaveOccurred())
				Expect(c.Server).To(Equal(expectedServer))
			})
		})
	})
})
