# 城通网盘下载工具

使用aria2来下载在城通网盘上分享的文件

**本工具仅支持城通网盘会员用户使用，不提供破解会员服务的功能**

## 使用例子

- `cookie`: 填写`400gb.com`的`pubcookie`（需登陆后）
- `concurrent`: 同时下载任务数
- `fileID`: 填写`https://545c.com/dir/`后面的字符串

```shell script
ct2aria.linux -cookie=${Cookie} -aria2-endpoint='http://127.0.0.1:6800/jsonrpc' -concurrent=3 ${fileID}
```

