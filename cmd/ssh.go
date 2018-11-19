// Copyright © 2018 Pratheek Hegde <ptk609@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/howeyc/gopass"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// VaultErrorResponse struct for error response from vault API
type VaultErrorResponse struct {
	Errors []string
}

// VaultAuthLoginResponse struct for success response from vault API after logging in
type VaultAuthLoginResponse struct {
	RequestID     string      `json:"request_id"`
	LeaseID       string      `json:"lease_id"`
	Renewable     bool        `json:"renewable"`
	LeaseDuration int         `json:"lease_duration"`
	Data          interface{} `json:"data"`
	WrapInfo      interface{} `json:"wrap_info"`
	Warnings      interface{} `json:"warnings"`
	Auth          struct {
		ClientToken   string   `json:"client_token"`
		Accessor      string   `json:"accessor"`
		Policies      []string `json:"policies"`
		TokenPolicies []string `json:"token_policies"`
		Metadata      struct {
			Username string `json:"username"`
		} `json:"metadata"`
		LeaseDuration int    `json:"lease_duration"`
		Renewable     bool   `json:"renewable"`
		EntityID      string `json:"entity_id"`
	} `json:"auth"`
}

// VaultSSHOTPResponse struct for valid otp response
type VaultSSHOTPResponse struct {
	LeaseID       string `json:"lease_id"`
	Renewable     bool   `json:"renewable"`
	LeaseDuration int    `json:"lease_duration"`
	Data          struct {
		IP       string `json:"ip"`
		Key      string `json:"key"`
		KeyType  string `json:"key_type"`
		Port     int    `json:"port"`
		Username string `json:"username"`
	} `json:"data"`
	Warnings interface{} `json:"warnings"`
	Auth     interface{} `json:"auth"`
}

var selectedServerNumber int
var vaultUserToken string
var sshOTPKey string

// sshCmd represents the ssh command
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Login to a Server with Vault OTP",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		showVaultLoginPrompt()
		showServerSelection()
		generateVaultCredentials()
		loginToServer()
	},
}

func showVaultLoginPrompt() {
	var vaultUsername string
	fmt.Print("Enter your Vault user name: ")
	fmt.Scanln(&vaultUsername)
	fmt.Print("Enter your Vault password: ")
	vaultPassword, _ := gopass.GetPasswd()
	log.Println("Logging into Vault...")
	payload := fmt.Sprintf(`{"password": "%s"}`, vaultPassword)
	body := strings.NewReader(payload)
	req, err := http.NewRequest("POST", cfg.VaultAddress+"/v1/auth/userpass/login/"+vaultUsername, body)
	if err != nil {
		// handle err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	responseBody, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}
	if resp.StatusCode == 200 {
		VaultAuthloginResponse := VaultAuthLoginResponse{}
		json.Unmarshal(responseBody, &VaultAuthloginResponse)
		log.Println("Logged into Vault...")
		vaultUserToken = VaultAuthloginResponse.Auth.ClientToken
	} else {
		VaulterrorResponse := VaultErrorResponse{}
		json.Unmarshal(responseBody, &VaulterrorResponse)
		fmt.Printf("Error: %s\n", VaulterrorResponse.Errors[0])
		os.Exit(1)
	}
}

func showServerSelection() {
	attempt := 1
	maxAttempt := 3

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Number", "Server Name", "IP"})
	table.SetCaption(true, "Enter the number of the server you want to log in. eg: 1")
	for key, s := range cfg.Servers {
		table.Append([]string{strconv.Itoa(key + 1), s.ServerName, s.IP})
	}
	table.Render() // Send output
	// get server number from the prompt
	for true {
		fmt.Scanln(&selectedServerNumber)
		if attempt == maxAttempt {
			println("Reached max invalid attempt", maxAttempt)
			os.Exit(1)
		}
		if selectedServerNumber < 1 || selectedServerNumber > len(cfg.Servers) {
			attempt++
			fmt.Printf("Please enter a valid number between %d and %d!\n", 1, len(cfg.Servers))
		} else {
			break
		}
	}
}
func generateVaultCredentials() {
	fmt.Println("You selected", selectedServerNumber)
	fmt.Println("Generating OTP from vault for", cfg.Servers[selectedServerNumber-1].ServerName, "...")

	payload := fmt.Sprintf(`{"ip": "%s"}`, cfg.Servers[selectedServerNumber-1].IP)
	body := strings.NewReader(payload)

	req, err := http.NewRequest("POST", cfg.VaultAddress+"/v1/ssh/creds/"+cfg.Servers[selectedServerNumber].VaultRole, body)
	if err != nil {
		// handle err
	}
	req.Header.Set("X-Vault-Token", vaultUserToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()
	responseBody, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}
	if resp.StatusCode == 200 {
		otpResponse := VaultSSHOTPResponse{}
		json.Unmarshal(responseBody, &otpResponse)
		sshOTPKey = otpResponse.Data.Key
		fmt.Println("Generated OTP for", cfg.Servers[selectedServerNumber-1].ServerName, "...")

	} else {
		VaulterrorResponse := VaultErrorResponse{}
		json.Unmarshal(responseBody, &VaulterrorResponse)
		fmt.Printf("Error: %s\n", VaulterrorResponse.Errors[0])
		os.Exit(1)
	}
}

func loginToServer() {
	binary, lookErr := exec.LookPath("ssh")
	if lookErr != nil {
		panic(lookErr)
	}
	args := []string{"ssh", cfg.Servers[selectedServerNumber-1].LoginUsername + "@" + cfg.Servers[selectedServerNumber-1].IP}
	env := os.Environ()
	execErr := syscall.Exec(binary, args, env)
	if execErr != nil {
		panic(execErr)
	}
}
func init() {
	rootCmd.AddCommand(sshCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sshCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sshCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
