echo 'nameserver 114.114.114.114' >/etc/resolv.conf
curl -o /etc/yum.repos.d/epel.repo http://mirrors.aliyun.com/repo/epel-7.repo
curl -o /etc/yum.repos.d/Centos-7.repo http://mirrors.aliyun.com/repo/Centos-7.repo
yum install -y epel-release && yum install net-tools wget nethogs ntp socat -y

timedatectl set-timezone "Asia/Shanghai"

sudo rm /etc/localtime
sudo ln -s /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
echo 'server ntp1.aliyun.com' >/etc/ntp.conf

systemctl disable chronyd.service
systemctl enable ntpd
systemctl start ntpd
systemctl status ntpd
ntpdate -u ntp1.aliyun.com
timedatectl
touch /tmp/crontab.bak
crontab /tmp/crontab.bak
# crontab -l >/tmp/crontab.bak
echo "*/1 * * * * /usr/sbin/ntpdate -u ntp1.aliyun.com | logger -t NTP" >>/tmp/crontab.bak
crontab /tmp/crontab.bak

sed -i "s/#UseDNS yes/UseDNS no/g" /etc/ssh/sshd_config
systemctl restart sshd

echo -e 'TYPE="Ethernet"
PROXY_METHOD=none
BROWSER_ONLY=no
BOOTPROTO=static
IPADDR=192.168.1.200
GATEWAY=192.168.1.1
DEFROUTE=yes
IPV4_FAILURE_FATAL=no
IPV6INIT=no
IPV6_AUTOCONF=yes
IPV6_DEFROUTE=yes
IPV6_FAILURE_FATAL=no
IPV6_ADDR_GEN_MODE=stable-privacy
NAME=ens33
UUID=533399a5-fa94-312e-b341-337d65d9e2ff
DEVICE=ens33
ONBOOT=yes
ONBOOn=no' >/etc/sysconfig/network-scripts/ifcfg-ens33
hostnamectl set-hostname dns

version=1.18

rm -rf /usr/local/go*
rm -rf ~/.go*
yum install wget vim -y
wget https://golang.google.cn/dl/go$version.linux-amd64.tar.gz
mkdir /usr/local/go$version
tar -xvf go$version.linux-amd64.tar.gz -C /usr/local/go$version --strip-components 1

mkdir -p ~/.go/{bin,src,pkg}

cat <<EOF >>/etc/profile

export GOROOT="/usr/local/go$version"
export GOPATH=\$HOME/go  #工作地址路径
export GOBIN=\$GOROOT/bin
export PATH=\$PATH:\$GOBIN
EOF
source /etc/profile
go version
go env
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct

yum install git -y

systemctl stop firewalld
systemctl disable firewalld
systemctl stop dnsmasq
systemctl disable dnsmasq

yum -y install redis
sed -i "s/127.0.0.1/0.0.0.0/g" /etc/redis.conf
sed -i "s/daemonize no/daemonize yes/g" /etc/redis.conf
systemctl enable redis
systemctl start redis

git clone https://github.com/ls-2018/landns.git
cd landns
go build .
mv landns /usr/bin/

cat >/etc/systemd/system/landns.service <<EOF
[Unit]
Description=landns
After=redis.target

[Service]
Type=notify
Restart=always
RestartSec=5s
LimitNOFILE=40000
TimeoutStartSec=0
ExecStart=/usr/bin/landns -v -u 114.114.114.114:53 --redis=127.0.0.1:6379 --redis-database=15

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable landns.service
sudo systemctl start landns.service
sudo systemctl status landns.service
