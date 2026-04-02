Status while running
```
$ curl http://aspire.lamac.cc:6680/status -H "Accept: text/plain"
Bebel GILBERTO-River Song (Grant Nelson Remix Radio Edit)
Radio FM
♫
```

Status if not running
```
$ curl http://aspire.lamac.cc:6680/status -H "Accept: text/plain"
Stopped
```

Play
```
$ curl 'http://aspire.lamac.cc:6680/command?radio=https://live.rtvs.sk:8001/FM_128.mp3&play' -H "Accept:text/plain"
playing
```

Stop
```
$ curl 'http://aspire.lamac.cc:6680/command?stop' -H "Accept: text/plain"
stopping
```

List radios
```
$ curl http://aspire.lamac.cc:6680/radio -H "Accept: text/plain"
https://live.rtvs.sk:8001/FM_128.mp3  Radio FM
http://stream.radioparadise.com/aac-320  Radio Paradise
```