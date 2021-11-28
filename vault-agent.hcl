pid_file = "./pidfile"

auto_auth {
mount_path = "auth/approle"
method "approle" {
config = {
  role_id_file_path = "./app-role-id"
}
}

sink {
  type = "file"
  wrap_ttl = "30m"
  config = {
    path = "./wrapped_token"
  }
}

sink {
type = "file"
config = {
  path = "./unwrapped_token"
  }
}
}

vault {
address = "http://vault:8200"
}
exit_after_auth = true