# pg-db-admin

This is a utility to administer postgres databases that are behind a firewall.

The published docker image runs with a lambda entrypoint.
Using a lambda that is on the same VPC as the database, this utility can ensure a database exists with a specific owner.
This utilizes AWS IAM to secure administration instead of using an SSH Tunnel or VPN.
This also limits the actions that a user can take, making it extremely hard to perform malicious commands.
