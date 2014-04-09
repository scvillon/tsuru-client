// Copyright 2014 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
tsuru is a command line tool for application developers.

It provides some commands that allow a developer to register
himself/herself, manage teams, apps and services.

Usage:

	% tsuru <command> [args]

The currently available commands are (grouped by subject):

	target            retrive the current tsuru server
	target-add        add a new named tsuru server target
	target-remove     remove a named target
	target-set        set a target for usage by name
	version           displays current tsuru version

	user-create       creates a new user
	user-remove       removes your user from tsuru server
	login             authenticates the user with tsuru server
	logout            finishes the session with tsuru server
	change-password   changes your password
	reset-password    redefines your password
	key-add           adds a public key to tsuru deploy server
	key-remove        removes a public key from tsuru deploy server

	team-create       creates a new team (adding the current user to it automatically)
	team-remove       removes a team from tsuru
	team-list         list teams that the user is member
	team-user-add     adds a user to a team
	team-user-remove  removes a user from a team

	platform-list     list available platforms
	app-create        creates an app
	app-remove        removes an app
	app-list          lists apps that the user has access (see app-grant and team-user-add)
	app-info          displays information about an app
	app-grant         allows a team to have access to an app
	app-revoke        revokes access to an app from a team
	unit-add          adds new units to an app
	unit-remove       remove units from an app
	log               shows log for an app
	run               runs a command in all units of an app
	restart           restarts the app's application server
	set-cname         defines a cname for an app
	unset-cname       unsets the cname from an app
	swap              swaps the router between two apps

	env-get           display environment variables for an app
	env-set           set environment variable(s) to an app
	env-unset         unset environment variable(s) from an app

	bind              binds an app to a service instance
	unbind            unbinds an app from a service instance

	service-list      list all services, and instances of each service
	service-add       creates a new instance of a service
	service-remove    removes a instance of a service
	service-status    checks the status of a service instance
	service-info      list instances of a service, and apps bound to each instance
	service-doc       displays documentation for a service

Use "tsuru help <command>" for more information about a command.


Guessing app names

In some app-related commands (app-remove, app-info, app-grant, app-revoke, log,
run, restart, env-get, env-set, env-unset, bind and unbind), there is an
optional parameter --app, used to specify the name of the app.

The --app parameter is optional, if omitted, tsuru will try to "guess" the name
of the app based in the configuration of the git repository. It will try to
find a remove labeled "tsuru", and parse its url.

For example, if the file ".git/config" in you git repository contains the
following remote declaration:

	[remote "tsuru"]
	url = git@tsuruhost.com:gopher.git
	fetch = +refs/heads/*:refs/remotes/tsuru/*

When you run "tsuru app-info" without specifying the app, tsuru would display
information for the app "gopher".


Managing remote tsuru server endpoints

Usage:

	% tsuru target
	% tsuru target-add <label> <address> [--set-current|-s]
	% tsuru target-set <label>
	% tsuru target-remove <label>

The target is the tsuru server to which all operations will be directed to.

With this set of commands you are be able to check the current target, add a new labeled target, set a target for usage,
list the added targets and remove a target, respectively.


Check current version

Usage:

	% tsuru version

This command returns the current version of tsuru command.


Create a user

Usage:

	% tsuru user-create <email>

user-create creates a user within tsuru remote server. It will ask for the
password before issue the request.


Remove your user from tsuru server

Usage:

	% tsuru user-remove

user-remove will remove currently authenticated user from remote tsuru server.
since there cannot exist any orphan teams, tsuru will refuse to remove a user
that is the last member of some team. if this is your case, make sure you
remove the team using "team-remove" before removing the user.


Authenticate within remote tsuru server

Usage:

	% tsuru login <email>

Login will ask for the password and check if the user is successfully
authenticated. If so, the token generated by the tsuru server will be stored in
${HOME}/.tsuru_token.

All tsuru actions require the user to be authenticated (except login and
user-create, obviously).


Logout from remote tsuru server

Usage:

	% tsuru logout

Logout will delete the token file and terminate the session within tsuru
server.


Change user's password

Usage:

	% tsuru change-password

change-password will change the password of the logged in user. It will ask for
the current password, the new and the confirmation.


Redefine user's password

Usage:

	% tsuru reset-password <email> [--token|-k <token>]

reset-password will redefine the user password. This process is composed by two steps:

	1. Token generation
	2. Password generation

In order to generate the token, users should run this command without the --token flag.
The token will be mailed to the user.

With the token in hand, the user can finally reset the password using the --token flag.
The new password will also be mailed to the user.`,


Add SSH public key to tsuru's git server

Usage:

	% tsuru key-add [${HOME}/.ssh/id_rsa.pub]

key-add sends your public key to tsuru's git server. By default, it will try
send a public RSA key, located at ${HOME}/.ssh/id_rsa.pub. If you want to send
other file, you can call it with the path to the file. For example:

	% tsuru key-add /etc/my-keys/id_dsa.pub

The key will be added to the current logged in user.


Remove SSH public key from tsuru's git server

Usage:

	% tsuru key-remove [${HOME}/.ssh/id_rsa.pub]

key-remove removes your public key from tsuru's git server. By default, it will
try to remove a key that match you public RSA key located at
${HOME}/.ssh/id_rsa.pub. If you want to remove a key located somewhere else,
you can pass it as parameter to key-remove:

	% tsuru key-remove /etc/my-keys/id_dsa.pub

The key will be removed from the current logged in user.


Create a new team for the user

Usage:

	% tsuru team-create <team-name>

team-create will create a team for the user. tsuru requires a user to be a
member of at least one team in order to create an app or a service instance.

When you create a team, you're automatically member of this team.


Remove a team from tsuru

Usage:

	% tsuru team-remove <team-name>

team-remove will remove a team from tsuru server. You're able to remove teams
that you're member of. A team that has access to any app cannot be removed.
Before removing a team, make sure it does not have access to any app (see
"app-grant" and "app-revoke" commands for details).


List teams that the user is member of

Usage:

	% tsuru team-list

team-list will list all teams that you are member of.


Add a user to a team

Usage:

	% tsuru team-user-add <team-name> <user@email>

team-user-add adds a user to a team. You need to be a member of the team to be
able to add another user to it.


Remove a user from a team

Usage:

	% tsuru team-user-remove <team-name> <user@email>

team-user-remove removes a user from a team. You need to be a member of the
team to be able to remove a user from it.

A team can never have 0 users. If you are the last member of a team, you can't
remove yourself from it.


Display the list of available platforms

Usage:

	% tsuru platform-list

platform-list lists the available platforms. All platforms displayed in this
list may be used to create new apps (see app-create).


Create an app

Usage:

	% tsuru app-create <app-name> <platform>

app-create will create a new app using the given name and platform. For tsuru,
a platform is a Juju charm. To check the available platforms, use the command
"platform-list".

In order to create an app, you need to be member of at least one team. All
teams that you are member (see "tsuru team-list") will be able to access the
app.


Remove an app

Usage:

	% tsuru app-remove [--app appname]

app-remove removes an app. If the app is bound to any service instance, all
binds will be removed before the app gets deleted (see "tsuru unbind"). You
need to be a member of a team that has access to the app to be able to remove
it (you are able to remove any app that you see in "tsuru app-list").

The --app flag is optional, see "Guessing app names" section for more details.


List apps that you have access to

Usage:

	% tsuru app-list

app-list will list all apps that you have access to. App access is controlled
by teams. If your team has access to an app, then you have access to it.


Display information about an app

Usage:

	% tsuru app-info [--app name]

app-info will display some informations about an specific app (its state,
platform, git repository, etc.). You need to be a member of a team that access
to the app to be able to see informations about it.

The --app flag is optional, see "Guessing app names" section for more details.


Allow a team to access an app

Usage:

	% tsuru app-grant <team-name> [--app appname]

app-grant will allow a team to access an app. You need to be a member of a team
that has access to the app to allow another team to access it.

The --app flag is optional, see "Guessing app names" section for more details.


Revoke from a team access to an app

Usage:

	% tsuru app-revoke <team-name> [--app appname]

app-revoke will revoke the permission to access an app from a team. You need to
have access to the app to revoke access from a team.

An app cannot be orphaned, so it will always have at least one authorized team.

The --app flag is optional, see "Guessing app names" section for more details.


Add new units to the app

Usage:

	% tsuru unit-add <# of units> [--app appname]

unit-add will add new units (instances) to an app. You need to have access to
the app to be able to add new units to it.

The --app flag is optional, see "Guessing app names" section for more details.


Remove units from the app

Usage:

	% tsuru unit-remove <# of units> [--app appname]

unit-remove will remove units (instances) from an app. You need to have access
to the app to be able to remove units from it.

The --app flag is optional, see "Guessing app names" section for more details.


See app's logs

Usage:

	% tsuru log [--app|-a appname] [--lines|-l numberOfLines] [--source|-s source] [--follow|-f]

Log will show log entries for an app. These logs are not related to the code of
the app itself, but to actions of the app in tsuru server (deployments,
restarts, etc.).

The --app flag is optional, see "Guessing app names" section for more details.
The --lines flag is optional and by default its value is 10.
The --source flag is optional.


Run an arbitrary command in the app machine

Usage:

	% tsuru run <command> [commandarg1] [commandarg2] ... [commandargn] [--app appname]

Run will run an arbitrary command in the app machine. Base directory for all
commands is the root of the app. For example, in a Django app, "tsuru run" may
show the following output:

	% tsuru run polls ls
	app.yaml
	brogui
	deploy
	foo
	__init__.py
	__init__.pyc
	main.go
	manage.py
	settings.py
	settings.pyc
	templates
	urls.py
	urls.pyc

The --app flag is optional, see "Guessing app names" section for more details.


Define a CNAME for the app

Usage:

	% tsuru set-cname <cname> [--app appname]

set-cname will define a CNAME for the app. It will not manage any DNS register,
it's up to the user to create the DNS register. Once the app contains a custom
CNAME, it will be displayed by "app-list" and "app-info".

The --app flag is optional, see "Guessing app names" section for more details.


Undefine the CNAME from the app

Usage:

	% tsuru unset-cname [--app appname]

unset-cname undoes the change that set-cname does. After unsetting the CNAME
from the app, "app-list" and "app-info" will display the internal, unfriendly
address that tsuru uses.

The --app flag is optional, see "Guessing app names" section for more details.


Restart the app's application server

Usage:

	% tsuru restart [--app appname]

Restart will restart the application server (as defined in Procfile) of the
application.

The --app flag is optional, see "Guessing app names" section for more details.


Display environment variables of an application

Usage:

	% tsuru env-get [--app appname] [variable-names]

env-get will display the name and the value of environment variables exported
in the application's environment. If none name is given, it will display the
value of all environment variables exported in the app via tsuru. It omits the
value of private environment variables (exported by service binding, see bind
command for more details). Examples of use:

	% tsuru env-get myapp MYSQL_DATABASE_NAME MYSQL_PASSWORD
	MYSQL_DATABASE_NAME=myapp_sql
	MYSQL_PASSWORD=*** (private variable)
	% tsuru env-get myapp
	MYSQL_DATABASE_NAME=myapp_sql
	MYSQL_USER=secret
	MYSQL_HOST=remote.mysql.com
	MYSQL_PASSWORD=*** (private variable)
	% tsuru env-get myapp SOMETHING_UNKNOWN

The first command retrieves only the specified variables, while the second
command retrieves all variables. In the last command, we ask for an undefined
variable, and env-get fails silently. All environment variable related commands
fail silently.

The --app flag is optional, see "Guessing app names" section for more details.


Define the value of one or more environment variables

Usage:

	% tsuru env-set <NAME_1=VALUE_1> [NAME_2=VALUE_2] ... [NAME_N=VALUE_N] [--app appname]

env-set will (re)define environment variables for your app.  You can specify
one or more environment variables to (re)define. env-set cannot redefine
private variables, and all variables defined using env-set will be public (its
value will be displayed in env-get). env-set does not restart the application
after exporting the variables, for doing that, see restart command. Examples of
use:

	% tsuru env-set myapp MYSQL_DATABASE_NAME=myapp_sql2 MYSQL_PASSWORD=1234
	% tsuru env-get myapp MYSQL_DATABASE_NAME MYSQL_PASSWORD
	MYSQL_DATABASE_NAME=myapp_sql
	MYSQL_PASSWORD=*** (private variable)

Notice that env-set will fail silently to redefine private variables.

The --app flag is optional, see "Guessing app names" section for more details.


Undefine an environment variable

Usage:

	% tsuru env-unset <NAME_1> [NAME_2] ... [NAME_N] [--app appname]

env-unset will undefine environments variables in your app.  You can specify
one or more environment variables to undefine.  env-unset cannot remove private
variables. Examples of use:

	% tsuru env-unset myapp MYSQL_DATABASE_NAME MYSQL_PASSWORD
	% tsuru env-get myapp MYSQL_DATABASE_NAME MYSQL_PASSWORD
	MYSQL_PASSWORD=*** (private variable)

Notice that env-unset will fail silently to undefine private variables.

The --app flag is optional, see "Guessing app names" section for more details.


Bind an application to a service instance

Usage:

	% tsuru bind <instance-name> [--app appname]

Bind will bind an application to a service instance (see service-add for more
details on how to create a service instance).

When binding an application to a service instance, tsuru will add new
environment variables to the app. All environment variables exported by bind
will be private (not accessible via env-get).

The --app flag is optional, see "Guessing app names" section for more details.


Unbind an application from a service instance

Usage:

	% tsuru unbind <instance-name> [--app appname]

Unbind will unbind an application from a service instance.  After unbinding,
the instance will not be available anymore.  For example, when unbinding an
application from a MySQL service, the app would lose access to the database.

The --app flag is optional, see "Guessing app names" section for more details.


List available services and instances

Usage:

	% tsuru service-list

service-list will retrieve and display a list of services that the user has
access to. If the user has any instance of services, it will be displayed by
this command too.


Swap the routing between two apps

Usage:

	% tsuru swap <app1> <app2>

swap will swap the routing between two apps enabling blue/green deploy, zero downtime and make the rollbacks easier.

Create a new service instance

Usage:

	% tsuru service-add <service-name> <instance-name>

service-add will create a new service instance. After listing services with
"service-list", you may want to create a new service instance.

Example of use:

	% tsuru service-list
	+----------+-----------+
	| Services | Instances |
	+----------+-----------+
	| mysql    |           |
	+----------+-----------+
	% tsuru service-add mysql newmysql
	Service successfully added.
	% tsuru service-list
	+----------+-----------+
	| Services | Instances |
	+----------+-----------+
	| mysql    | newmysql  |
	+----------+-----------+


Remove a service instance

Usage:

	% tsuru service-remove <instance-name>

service-remove will destroy a service instance. It can't remove a service
instance that is bound to an app, so before remove a service instance, make
sure there is no apps bound to it (see "service-info" command).


Display information about a service

Usage:

	% tsuru service-info <service-name>

service-info will display a list of all instances of a given service (that the
user has access to), and apps bound to these instances.

Example of use:

	% tsuru service-info mysql
	Info for "mysql"
	+-----------+-------+
	| Instances | Apps  |
	+-----------+-------+
	| newmysql  |       |
	+-----------+-------+
	% tsuru bind newmysql myapp
	...
	% tsuru service-info mysql
	Info for "mysql"
	+-----------+-------+
	| Instances | Apps  |
	+-----------+-------+
	| newmysql  | myapp |
	+-----------+-------+


Check if a service instance is up

Usage:

	% tsuru service-status <instance-name>

service-status will display the status of the given service instance. For now,
it checks only if the instance is "up" (receiving connections) or "down"
(refusing connections).


Display the documentation of a service

Usage:

	% tsuru service-doc <service-name>

service-doc will display the documentation of a service.
*/
package main
