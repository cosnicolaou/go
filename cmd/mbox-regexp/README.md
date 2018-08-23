mbox-regexp
===========

Command mbox-regexp runs a regexp (using go's regexp.Expand call) with selective expansion
against an mboxrd format mailbox file, such as those produced by gmail exports.exports

The invocation below will print out all of the URL's in the email messages.
```
./mbox-regexp --pattern='(?U)<https://(?P<url>.*)>' --template='$url' Takeout/your-mbox-file
```

mbox-regexp correctly handles multi-part mime format emails and applies the regexp against
parts that are in quotedprintable encoding.
