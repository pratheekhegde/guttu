Guttu
===

A CLI tool for simplifying HashiCorp Vault's One-Time SSH Passwords logins. 

### Install

```go get github.com/pratheekhegde/guttu```

### Configuration

Sample `.guttu.yaml` configuration file. `guttu` will search for this configuration file in your home directory.

```
vault_address: https://w.x.y.z:8200
servers:
- ip: x.x.x.x
  server_name: staging-app-server
  login_username: ubuntu
  vault_role: staging-app-server-role
- ip: x.x.x.x
  server_name: prod-app-server
  login_username: ubuntu
  vault_role: prod-app-server-role
- ip: x.x.x.x
  server_name: staging-web-server
  login_username: ubuntu
  vault_role: staging-web-server-role
  ```