# pine

pine is a 9P proxy to inspect 9P dialogs

Written to avoid adding chatty options to all clients and servers and
as an experiment. Example usage (shell in use is rc, 9p program comes
from plan9port):

	% pine -remote tcp!localhost!3333 -local unix!test.sock >test.out >[2]test.err &
	% 9p -a test.sock ls
	... output omitted ...
	% cat test.out
	Tversion tag 65535 msize 8192 version '9P2000'
	Rversion tag 65535 msize 8192 version '9P2000'
	Tauth tag 0 afid 0 uname 'nicolagi' nuname 4294967295 aname ''
	Rerror tag 0 ename 'no authentication required' ecode 0
	Tattach tag 0 fid 0 afid 4294967295 uname 'nicolagi' nuname 4294967295 aname ''
	Rattach tag 0 aqid (dc76e9f0c0006e8f aefd4576 'd')
	Twalk tag 0 fid 0 newfid 1
	Rwalk tag 0
	Tstat tag 0 fid 1
	Rstat tag 0 st ('root' 'nicolagi' '' '' q (dc76e9f0c0006e8f aefd4576 'd') m d700 at 1562737600 mt 1561455586 l 0 t 0 d 0 ext )
	Tclunk tag 0 fid 1
	Rclunk tag 0
	Twalk tag 0 fid 0 newfid 1
	Rwalk tag 0
	Topen tag 0 fid 1 mode 0
	Ropen tag 0 qid (dc76e9f0c0006e8f aefd4576 'd') iounit 0
	Tread tag 0 fid 1 offset 0 count 8168
	Rread tag 0 count 1667
	Tread tag 0 fid 1 offset 1667 count 8168
	Rread tag 0 count 0
	Tclunk tag 0 fid 1
	Rclunk tag 0
	% cat test.err | jq .
	{
	  "in": {
	    "Name": "@",
	    "Net": "unix"
	  },
	  "level": "info",
	  "msg": "Starting net pipe",
	  "op": "pipe",
	  "out": {
	    "IP": "127.0.0.1",
	    "Port": 57250,
	    "Zone": ""
	  },
	  "time": "2019-07-10T06:46:40+01:00"
	}
	{
	  "in": {
	    "IP": "127.0.0.1",
	    "Port": 3333,
	    "Zone": ""
	  },
	  "level": "info",
	  "msg": "Starting net pipe",
	  "op": "pipe",
	  "out": {
	    "Name": "test.sock",
	    "Net": "unix"
	  },
	  "time": "2019-07-10T06:46:40+01:00"
	}

The project name means nothing, just a random noun that occurred to me.
