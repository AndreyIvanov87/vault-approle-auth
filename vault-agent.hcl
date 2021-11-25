pid_file = "./pidfile"

auto_auth {
  mount_path = "auth/approle"
  method "approle" {
    config = {
      role_id_file_path = "./roleID"
      secret_id_response_wrapping_path = "auth/approle/role/go_webapp/secret-id"
    }
  }

  sink {
      type = "file"
      wrap_ttl = "30m"
      config = {
        path = "wrapped_token"
      }
    }

  sink {
    type = "file"
    config = {
      path = "unwrapped_token"
      }
    }
}

vault {
  address = "https://127.0.0.1:8200"
}
