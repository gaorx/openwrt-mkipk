# Mkipk - Openwrt ipk 打包工具

使用方法:

1. 将要打包的目录按如下布局放入目录中，例如:

```
/home/gaorx/tmp/RexGatewayApp
├── control
│   ├── control
│   └── postinst
└── data
    ├── app
    │   ├── BackupPack
    │   ├── DeviceOta
    │   ├── GateWayDaemon.bin
    │   ├── ObjectModelList
    │   │   └── RexGateWayModel.ini
    │   ├── RexGateWayApp
    │   ├── ScenesModelList
    │   ├── SubDevicesList
    │   └── ca
    │       ├── client.crt
    │       ├── client.key
    │       └── root-ca.crt
    ├── etc
    │   ├── rc.local.back
    │   └── sysctl.conf.back
    └── usr
        └── lib
            ├── libcrypto.so.1.1
            ├── libcurl.so.4
            ├── libpaho-mqtt3c.so.1
            ├── libpaho-mqtt3cs.so.1
            ├── librexgatewaysdk.so
            └── libssl.so.1.1
```

2. 执行命令：

```bash
# 需将输入和输出换成自己的路径
go run mkipk.go -i /path/to/input_dir -o /path/to/output_dir
```

3. 执行完成后，在`/path/to/output_dir`中就可以看到输出的ipk文件了
