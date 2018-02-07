# json_encode

json_encode package pipes input into JSON for further processing.

## Installation

golang 1.9+ should be installed. The environment installation is very simple : https://golang.org/doc/install

To compile :

    git clone https://github.com/fedir/json_encode.git
    cd json_encode
    go build

After put compiled `json_encode` in Your favourite bin folder.

## Usage

    echo "Hello world" | ./json_encode
    {"0":"Hello world"}

    ls /usr/local/go/src/ | ./json_encode
    {"0":"Make.dist","1":"all.bash","10":"bytes","11":"clean.bash","12":"clean.bat","13":"clean.rc","14":"cmd","15":"cmp.bash","16":"compress","17":"container","18":"context","19":"crypto","2":"all.bat","20":"database","21":"debug","22":"encoding","23":"errors","24":"expvar","25":"flag","26":"fmt","27":"go","28":"hash","29":"html","3":"all.rc","30":"image","31":"index","32":"internal","33":"io","34":"iostest.bash","35":"log","36":"make.bash","37":"make.bat","38":"make.rc","39":"math","4":"androidtest.bash","40":"mime","41":"naclmake.bash","42":"nacltest.bash","43":"net","44":"os","45":"path","46":"plugin","47":"race.bash","48":"race.bat","49":"reflect","5":"archive","50":"regexp","51":"run.bash","52":"run.bat","53":"run.rc","54":"runtime","55":"sort","56":"strconv","57":"strings","58":"sync","59":"syscall","6":"bootstrap.bash","60":"testing","61":"text","62":"time","63":"unicode","64":"unsafe","65":"vendor","7":"bufio","8":"buildall.bash","9":"builtin"}

    tail /var/log/apache2/access_log | ./json_encode
    {"0":"127.0.0.1 - - [04/Nov/2017:00:40:50 +0100] \"GET / HTTP/1.1\" 304 -","1":"127.0.0.1 - - [04/Nov/2017:00:40:51 +0100] \"GET / HTTP/1.1\" 304 -","2":"127.0.0.1 - - [04/Nov/2017:00:40:51 +0100] \"GET / HTTP/1.1\" 304 -","3":"127.0.0.1 - - [04/Nov/2017:00:40:52 +0100] \"GET / HTTP/1.1\" 304 -","4":"127.0.0.1 - - [04/Nov/2017:00:40:52 +0100] \"GET / HTTP/1.1\" 304 -","5":"127.0.0.1 - - [04/Nov/2017:00:40:53 +0100] \"GET / HTTP/1.1\" 304-","6":"127.0.0.1 - - [04/Nov/2017:00:40:54 +0100] \"GET / HTTP/1.1\" 304 -","7":"127.0.0.1 - - [04/Nov/2017:00:40:54 +0100] \"GET / HTTP/1.1\" 304 -","8":"127.0.0.1 - - [04/Nov/2017:00:57:57 +0100] \"-\" 408 -","9":"127.0.0.1 - - [04/Nov/2017:00:57:57 +0100] \"-\" 408 -"}
