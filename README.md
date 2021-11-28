# AppRole Auth Method Go Example

Метод авторизации approle позволяет компьютерам или приложениям проходить аутентификацию с помощью ролей, определенных Vault.
Этот метод аутентификации ориентирован на автоматизированные рабочие процессы (машины и сервисы) и менее полезен для людей-операторов.

«AppRole» представляет собой набор политик Vault и ограничений входа в систему, которые должны быть соблюдены для получения токена с этими политиками. 
Объем может быть как узким, так и широким по желанию. AppRole может быть создан для конкретной машины, или даже для конкретного пользователя на этой машине,
или для службы, распределенной по машинам. Учетные данные, необходимые для успешного входа в систему, зависят от ограничений, установленных для AppRole,
связанного с учетными данными.

## Конфигурация Vault

Методы аутентификации должны быть настроены заранее, прежде чем пользователи или машины смогут аутентифицироваться. 
Эти шаги обычно выполняются оператором или инструментом управления конфигурацией.

### Подготовка
Для конфигурирования данного метода авторизации можно использовать root token. Что само по себе не безопасно и не рекомендуется.
Хорошей практикой создавать отдельные учетные записи для каждого инженереа, или например включить авторизацию через LDAP.

#### У пользователя выполняющего дальнейшие действия должна быть привязана следующая политика:

```
# Mount the AppRole auth method
path "sys/auth/approle" {
  capabilities = [ "create", "read", "update", "delete", "sudo" ]
}

# Configure the AppRole auth method
path "sys/auth/approle/*" {
  capabilities = [ "create", "read", "update", "delete" ]
}

# Create and manage roles
path "auth/approle/*" {
  capabilities = [ "create", "read", "update", "delete", "list" ]
}

# Write ACL policies
path "sys/policies/acl/*" {
  capabilities = [ "create", "read", "update", "delete", "list" ]
}

# Write test data
# Set the path to "secret/data/mysql/*" if you are running `kv-v2`
path "secret/mysql/*" {
  capabilities = [ "create", "read", "update", "delete", "list" ]
}
```

#### Настроить подключение к нужному инстансу Vault

```shell
$ export VAULT_ADDR=http://127.0.0.1:8200
$ export VAULT_TOKEN=your_token_here
```

#### Добавить секретные данные используя cli если необходимо, или же добавьте нужные данные через GUI

```shell
$ vault kv put secrets/k11s/demo/app/service db_name="users" username="admin" password="passw0rd"
```

### Настройка

#### 1. Включить метод аутентификации
```shell
$ vault auth enable approle
```
#### 2. Создать политику для роли

```
# Read-only permission on secrets stored at 'secret/data/mysql/webapp'
path "secret/data/mysql/webapp" {
  capabilities = [ "read" ]
}
```

```shell
$ vault policy write -tls-skip-verify app_policy_name -<<EOF
# Read-only permission on secrets stored at 'secrets/k11s/demo/app/service'
path "secrets/data/k11s/demo/app/service" {
  capabilities = [ "read" ]
}
EOF

```
#### 3. Создать роль

```shell
$ vault write -tls-skip-verify auth/approle/role/my-app-role \
  token_policies="app_policy_name" \
  token_ttl=1h \
  token_max_ttl=4h \
  secret_id_bound_cidrs="0.0.0.0/0","127.0.0.1/32" \
  token_bound_cidrs="0.0.0.0/0","127.0.0.1/32" \
  secret_id_ttl=60m policies="app_policy_name" \
  bind_secret_id=false
```

#### 4. Проверить созданную роль

```shell
$ vault read -tls-skip-verify auth/approle/role/my-app-role
```

#### 5. Получить RoleID

```shell
$ vault read -tls-skip-verify auth/approle/role/my-app-role/role-id
```

**RoleID** - это идентификатор, который выбирает AppRole, по которому оцениваются другие учетные данные. 
При аутентификации по конечной точке входа в систему этого метода аутентификации RoleID всегда является обязательным аргументом (через role_id). 
По умолчанию RoleID - это уникальные UUID, которые позволяют им служить вторичными секретами для другой информации об учетных данных. 
Однако они могут быть установлены на определенные значения, чтобы соответствовать интроспективной информации клиента (например, доменному имени клиента).

> role_id - обязательные учетные данные в конечной точке входа. Для AppRole, на который указывает role_id будут наложены ограничения. 

> bind_secret_id - Требует обязательно или нет предоставлять secret_id в точке регистрации. Если значение будет true то Vault Agent не сможет авторизоваться.

Дополнительно можно настроить и другие ограничени для AppRole.
Например, secret_id_bound_cidrs будет разрешено входить в систему только с IP-адресов, принадлежащих настроенным блокам CIDR на AppRole.

Документацию по AppRole API можно прочесть [тут.](https://www.vaultproject.io/api/auth/approle)

## Использование

### Вариант 1

RoleID эквивалентен имени пользователя, а SecretID - соответствующему паролю. Приложению необходимо и то, и другое для входа в Vault.
Естественно, следующий вопрос заключается в том, как безопасно доставить эти секреты клиенту.

Например, Ansible может использоваться как доверенный способ доставки RoleID на виртуальную машину.
Когда приложение запускается на виртуальной машине, RoleID уже существует на виртуальной машине.

![Vault AppRole Auth Diagram!](/img/vault-auth-basic-2.png "Vault AppRole Auth Diagram")

#### Для получения wrap token, который будет использоваться при авторизации и запросе secret-id выполните команду:

```shell
$ vault write -wrap-ttl=600s -tls-skip-verify -force auth/approle/role/my-app-role/secret-id
```
Затем полученный файл необходимо поместить на файловую систему виртуальной машины, где запускается наше приложение.
Этот путь должен совпадать с тем, которое ожидает приложение. **Полученный токен может быть использован лишь один раз.** 
После запуска, токен будет прочитан и использован для получения secret-id. 
После получения secret-id приложение может запросить данные из секретного хранилища Vault пока не истек TTL для secret_id.
Данный способ хорошо подходит в том случа, когда мы заполняем начальную конфигурацию приложения на этапе запуска. 
Например применив паттерн Singleton заполняем значениями структуру Config.

Рекомендации гласят что wrap token должен иметь время жизни равное времени деплоя приложения.
В то время secret-id могут иметь длинное время жизни, но это не рекомендуется производителем в силу большого потребления
памяти и нагрузке при очистке истекших токенов. Лучшая рекомендация гласит, что нужно использовать короткое время жизни
и постоянно продлевать его. Тем самым избегат дорогих операций авторизации и выдачи новых токенов.

### Вариант 2

Для передачи secret-id можно использовать vault-agent, который позволяет зная только role-id получить secret-id из хранилища.
При этом агент берет на себя функции по обновлению данного токена. Агент может поставлять как wrap token так и готовый 
к использованию secret-id.

#### Подготовка конфигурации агента vault-agent.hcl

```
pid_file = "./pidfile"

auto_auth {
  mount_path = "auth/approle"
  method "approle" {
    config = {
      role_id_file_path = "./roleID"
      secret_id_response_wrapping_path = "auth/approle/role/my-app-role/secret-id"
    }
  }

  sink {
    type = "file"
    wrap_ttl = "30m"
    config = {
      path = "./token_wrapped"
      }
    }

  sink {
    type = "file"
    config = {
      path = "./token_unwrapped"
      }
    }
}

vault {
  address = "https://127.0.0.1:8200"
}

```

#### Файл c role-id roleID
```text
d6194f18-5419-af2d-0c19-17aea4ba0378
```

#### Запуск Vault Agent

```shell
$ vault agent -tls-skip-verify -config=vault-agent.hcl -log-level=debug
```

После запуска агента на файловой системе появятся два файла согласно нашей конфигурации.
Первый файл будет в json формате и содержать данные wrap token для получения secret-id.
Второй файл будет содержать secret-id готовый к применению для авторизации.
Агент берет на себя функционал по обновлению ланных токенов.

Применение такой конфигурации оправданно когда наше приложение постоянно читает или пишет данные в хранилище Vault.

## Запуск приложения

### Для сборки приложения можно использовать прилагаемый Makefile.

#### Запуск сборки для Linux производится командой:

```shell script
$ make build-linux
```

#### Сборка для Mac OS:

```shell script
$ make build
```

#### Запуск без сборки для MacOS
```shell script
$ make run
```

#### Запуск без сборки для Linux

```shell script
$ make run-linux
```