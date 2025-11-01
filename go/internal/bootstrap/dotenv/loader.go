//go:build !prod

package dotenv

func init() {
	loadDotenv()
}
