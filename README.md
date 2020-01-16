# ConoHa Net

[ConoHa](https://www.conoha.jp/)のセキュリティグループを管理するためのツールです。

ConoHaには仮想マシン(VPS)が繋がっている仮想スイッチ側にパケットフィルタが備わっています。VPS上のOSのファイアーウォール機能(iptables, firewalld)に影響されずに利用できるので便利ですが、[API](https://www.conoha.jp/docs/)でしか操作することができません。このツールはそれらをコマンドラインで操作できるようにするものです。

[OpenStackのコマンドラインツール](http://docs.openstack.org/cli-reference/)でも同じことができますが、機能を絞っている分```conoha-net```の方が簡単に使えると思います。

## できること

* 通信方向(Ingress / Egress)
* プロトコルの種類(TCP / UDP / ICMP)
* プロトコルのバージョン(IPv4 / IPv6)
* 接続元IPアドレス,もしくはIPレンジ

以上の組み合わせによるパケットフィルタリング。

## インストール

任意のディレクトリに実行ファイルをダウンロードして実行して下さい。

**Mac OSX**

```shell
curl -sL https://github.com/hironobu-s/conoha-net/releases/download/current/conoha-net-osx.amd64.gz | zcat > conoha-net && chmod +x ./conoha-net
```


**Linux(amd64)**

```bash
curl -sL https://github.com/hironobu-s/conoha-net/releases/download/current/conoha-net-linux.amd64.gz | zcat > conoha-net && chmod +x ./conoha-net
```

**Windows(amd64)**

[ZIP file](https://github.com/hironobu-s/conoha-net/releases/download/current/conoha-net.amd64.zip)

## 使い方

例として、mygroupと言う名前のセキュリティグループを作成して、133.130.0.0/16からTCP 22番ポート宛の通信のみを許可するルールを作成してみます。

### 1. 認証

conoha-netを実行するには、APIの認証情報を環境変数にセットする必要があります。

I認証情報は「APIユーザ名」「APIパスワード」「テナント名 or テナントID」です。これらの情報は[ConoHaのコントロールパネル](https://manage.conoha.jp/API/)にあります。

以下はbashの例です。

```shell
export OS_USERNAME=[username]
export OS_PASSWORD=[password]
export OS_TENANT_NAME=[tenant name]
export OS_AUTH_URL=[identity endpoint]
export OS_REGION_NAME=[region]
```

参考: https://wiki.openstack.org/wiki/OpenStackClient/Authentication


### 2. セキュリティグループを作成する

create-groupで**my-group**と言う名前のセキュリティグループを作成します。

```
conoha-net create-group conoha-net
```

list-groupを実行すると、今作ったセキュリティグループが表示されます。

```
# conoha-net list-group
UUID                                     SecurityGroup     Direction     EtherType     Proto     IP Range     Port
05bb817c-5179-4156-99ec-f088ff5c5d8e     my-group          egress        IPv6          ALL                    ALL
5ecc4a23-0b92-4394-bca6-2466f08ef45e     my-group          egress        IPv4          ALL                    ALL
```


### 2. ルールを作成する

セキュリティグループにルールを追加することで、フィルタリングの挙動を設定します。これはcreate-ruleで行います。オプションは下記です。

```
OPTIONS:
   -d value, --direction value         (Required) The direction in which the rule applied. Must be either "ingress" or "egress" (default: "ingress")
   -e value, --ether-type value        (Required) Type of IP version. Must be either "Ipv4" or "Ipv6". (default: "IPv4")
   -p value, --port-range value        The source port or port range. For example "80", "80-8080".
   -P value, --protocol value          The IP protocol. Valid value are "tcp", "udp", "icmp" or "all". (default: "all")
   -g value, --remote-group-id value   The remote group ID to be associated with this rule.
   -i value, --remote-ip-prefix value  The IP prefix to be associated with this rule.
```

たとえば、133.130.0.0/16のIPレンジからのTCP 22番ポートへのインバウンド通信(ingress)を許可する場合は以下のように設定します。(-dオプションと-eオプションはデフォルト値があるので省略可能です)

```
conoha-net create-rule -d ingress -e IPv4 -p 22 -P tcp -i 133.130.0.0/16 my-group
```

再度list-groupを実行すると、ルールが追加されていることが確認できます。

```shell
UUID                                     SecurityGroup     Direction     EtherType     Proto     IP Range           Port
05bb817c-5179-4156-99ec-f088ff5c5d8e     my-group          egress        IPv6          ALL                          ALL
5ecc4a23-0b92-4394-bca6-2466f08ef45e     my-group          egress        IPv4          ALL                          ALL
83e287b1-1bcd-425c-b162-8b2d5e008ddf     my-group          ingress       IPv4          tcp       133.130.0.0/16     22 - 22
```

### 3. VPSにアタッチする

作成したセキュリティグループを一つ、もしくは複数のVPSにアタッチすることで、そのVPSに対してフィルタリングが有効になります。これにはattachを使います。

```shell
conoha-net attach -n [VPS名] my-group
```

-n はVPSを名前で指定します。他に-i(IPアドレスで指定), --id(UUIDで指定)も利用可能です。最後の引数は作成したセキュリティグループ名です。

listを実行すると、VPSにセキュリティグループがアタッチされたことを確認できます。

```
# conoha-net list
NameTag          IPv4               IPv6                                  SecGroup
Hironobu-test    163.44.***.***     2400:8500:1302:810:163:44:***:***     default, my-group
```

## コマンド一覧

-hオプションでヘルプが表示されます。

```shell
NAME:
ConoHa Net - Security group management tool for ConoHa

USAGE:
commands [global options] command [command options] [arguments...]

VERSION:
0.1

COMMANDS:
list          list all VPS
attach        attach a security group to VPS
detach        dettach a security group from VPS
list-group    list security groups and rules
create-group  create a security group
delete-group  delete a security group
create-rule   create a security group rule
delete-rule   delete a security group rule

GLOBAL OPTIONS:
--debug, -d    print debug informations.
--output value, -o value  specify output type. must be either "text" or "json". (default: "text")
--help, -h     show help
--version, -v  print the version
```

## (注意)あらかじめConoHa側で用意されているセキュリティグループについて

ConoHaには標準で下記のセキュリティグループが用意されています。これらはVPSへのアタッチ/デタッチは自由にできますが、変更/削除はできないようになっています。また**defaultはアタッチしないと全ての通信が通らなくなる**ので、事実上アタッチが必須となります。

* default
* gncs-ipv4-all
* gncs-ipv4-ssh
* gncs-ipv4-web
* gncs-ipv6-all

conoha-netのセキュリティグループを一覧表示するコマンドlist-groupは、デフォルトで**これらを表示しません**。--allオプションを明示的に指定する必要があります。

## ライセンス

MIT
