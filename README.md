# Warning
This project is a proof of concept and is not production tested.  DO NOT use it by import from here.  Rather copy it into your project and modify and test to meet your needs until you are confident in it before using it.  Maybe someday that will change but for now, you are warned, this project may be moved or deleted or who knows what.

# TokenSync
> Why does this package exist?

Good question.  We needed 4 k8s pods to have regular access to a 3rd party API which only allowed access via access token.  Each time you request a new token it invalidates the previous so each pod can not have different tokens which are valid at the same time.

This package was created as a proof of concept of how we could avoid race conditions and share a single token across many pods.

As of the time of my writing this, this package is a WIP and not complete.