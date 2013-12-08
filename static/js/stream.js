var logs = document.getElementById('logs');
var es = new EventSource('/stream');
es.onmessage = function(ev) {
    console.log(ev);

    var html = ev.data;

    var log = document.createElement('div');
    log.innerHTML =  html + "\n";

    while (log.firstChild) {
        logs.appendChild( log.firstChild );
    }
};

