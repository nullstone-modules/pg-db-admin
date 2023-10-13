# pg-db-admin

This is a utility to administer postgres databases that are behind a firewall.

Using a lambda that is on the same VPC as the database, this utility can ensure a database exists with a specific owner.
This utilizes AWS IAM to secure administration instead of using an SSH Tunnel or VPN.
This also limits the actions that a user can take, making it extremely hard to perform malicious commands.

## AWS Lambda setup

The Lambda requires specific configuration to work properly:

- A SecretsManager Secret containing the connection string as a postgres URL.
- `DB_CONN_URL_SECRET_ID` env var containing ARN of the AWS SecretsManager Secret.
- The execution role must have access to the above secret.
- The executing lambda must have network access to the postgres cluster.

## How it works

There are 3 actions that the AWS code performs to grant database access:
- `create-database`
- `create-user`
- `create-db-access`

### `create-database`

This action performs the following steps:
1. Ensures that a new user exists whose role name is `databaseName`.
2. Ensures that a database with the injected `databaseName` exists.
3. The newly-created database has an owner of the `databaseName` role.

### `create-user`

This action performs the following steps:
1. Ensure the user `username` exists.
2. If `username` role already exists, set the password to `password`.

### `create-db-access`

This action performs the following steps:
1. Add `username` as a member to the owner of the database.
2. Alters `username` so that the database owner has access to any schema objects created by `username`.
3. Grant all privileges on the `databaseName` and the `public` schema in `databaseName`.

## In Practice

In practice, the following should be true.

1. An application role runs migrations to create and alter schema objects.
2. Implicitly, this application role owns newly-created schema objects.
3. All application roles are a member of the role that owns the database -- giving them implicit access to all schema objects.
4. The database owner role is given access to all schema objects (present and future).

It's important to note that an application user created for a worker application typically does not perform migrations.
This application user is granted access to schema objects because it has membership in the database owner role (which has explicit access to schema objects).

## Repair database

In early versions of this module (below v0.2.0), schema objects were created and managed differently.
Your database may be left in a bad state.
To fix, follow these steps:
1. Set the database owner to a role with the same name as the database.
2. Ensure all application roles have membership to the database owner role.
3. Alter default privileges `FOR ROLE <application-role>` `TO <database-owner-role>`.
4. Grant privileges to all schema objects to application role on database and schema.
5. Set ownership of tables to any application role.

### Example access privilege outputs

```shell
oracle-> \dp
                                          Access privileges
 Schema |         Name          | Type  |      Access privileges      | Column privileges | Policies 
--------+-----------------------+-------+-----------------------------+-------------------+----------
 public | expiring_downloads    | table | postgres0=arwdDxt/postgres0+|                   | 
        |                       |       | oracle=arwdDxt/postgres0    |                   | 
 public | flyway_schema_history  | table | postgres0=arwdDxt/postgres0+|                   | 
        |                       |       | oracle=arwdDxt/postgres0    |                   | 
 public | module_artifacts      | table | postgres0=arwdDxt/postgres0+|                   | 
        |                       |       | oracle=arwdDxt/postgres0    |                   | 
 public | module_versions       | table | postgres0=arwdDxt/postgres0+|                   | 
        |                       |       | oracle=arwdDxt/postgres0    |                   | 
 public | modules               | table | postgres0=arwdDxt/postgres0+|                   | 
        |                       |       | oracle=arwdDxt/postgres0    |                   | 
(5 rows)

oracle-> \ddp
                        Default access privileges
    Owner     | Schema |   Type   |           Access privileges           
--------------+--------+----------+---------------------------------------
 oracle-zshgw |        | function | =X/"oracle-zshgw"                    +
              |        |          | oracle=X/"oracle-zshgw"              +
              |        |          | "oracle-zshgw"=X/"oracle-zshgw"
 oracle-zshgw |        | schema   | oracle=UC/"oracle-zshgw"             +
              |        |          | "oracle-zshgw"=UC/"oracle-zshgw"
 oracle-zshgw |        | sequence | oracle=rwU/"oracle-zshgw"            +
              |        |          | "oracle-zshgw"=rwU/"oracle-zshgw"
 oracle-zshgw |        | table    | oracle=arwdDxt/"oracle-zshgw"        +
              |        |          | "oracle-zshgw"=arwdDxt/"oracle-zshgw"
 oracle-zshgw |        | type     | =U/"oracle-zshgw"                    +
              |        |          | oracle=U/"oracle-zshgw"              +
              |        |          | "oracle-zshgw"=U/"oracle-zshgw"
(5 rows)
```
