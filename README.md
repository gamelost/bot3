bot3
====

IRC facing end of bot3

## To Run Your Bot on an IRC Channel

Install nsq, either by compiling or by downloading from
`http://nsq.io/deployment/installing.html` and copying the files in the `bin`
directory to a directory in your `$PATH`, a suggestion is `/usr/local/bin/` or
creating `$HOME/.bin` and putting the extracted files in there.

It is recommended to use a terminal multiplexer of some sort, otherwise open
up as many terminal windows as needed:

    # Terminal 1
    nsqlookupd
    # Terminal 2
    nsqd --lookupd-tcp-address 127.0.0.1:4160

Make sure `$GOPATH` is set and you have go installed.

To install with Homebrew:

    brew install go

Run the following commands:

    # Terminal 3
    go get github.com/gamelost/bot3
    cd $GOPATH/src/github.com/gamelost/bot3
    go get ./...
    # Now get your config file up
    cp bot3.config{.default,}
    bot3

    # Terminal 4
    go get github.com/gamelost/bot3server
    cd $GOPATH/src/github.com/gamelost/bot3server
    go get ./...
    # Now get your config file up
    cp bot3server.config{.default,}
    bot3server

Your bot should now be running.

The default `bot3.config` setting put your bot in the `#tpsreport` channel.
To change it to point to your desired channel, edit the `bot3.config` file and
change the line `channel=tpsreport`. You may also want to change the default
nick for the bot from `glcbot3` to something else.
