# minemiter

The `minemiter` command processes the current `minigraph` through the
given set of templates, the result being a new file containing output
from the templates.

By default, `minemiter` uses the templates in the `templates` directory,
which generate `minimega` commands, and outputs the results to
`minemiter.mm`.

## Template Processing

When processed by `minemiter`, templates are grouped by each letter of
the alphabet, and each node is run through the group of templates
sequentially before going on to the next group. For example, let´s say
you have the following nodes and templates.

```
Nodes:

1, 2, 3

Templates:

A10
A11

M10

Z10
Z11
```

The processing order would be like the following.

```
Node 1 gets processed by templates A10 and A11
Node 2 gets processed by templates A10 and A11
Node 3 gets processed by templates A10 and A11

Node 1 gets processed by template M10
Node 2 gets processed by template M10
Node 3 gets processed by template M10

Node 1 gets processed by templates Z10 and Z11
Node 2 gets processed by templates Z10 and Z11
Node 3 gets processed by templates Z10 and Z11
```

Now, let´s say that template M10 contains an `{{ if once }}` statement.
In this case, only Node 1 would get processed by the M10 template.

```
Node 1 gets processed by templates A10 and A11
Node 2 gets processed by templates A10 and A11
Node 3 gets processed by templates A10 and A11

Node 1 gets processed by template M10

Node 1 gets processed by templates Z10 and Z11
Node 2 gets processed by templates Z10 and Z11
Node 3 gets processed by templates Z10 and Z11
```

Alternatively, let´s say that template Z10 contains a `{{ stop }}`
statement in it, but conditional only for Node 2. In this case, Node 2
doesn´t get processed by template Z11, but the other nodes do. The point
here is that the stop statement only halts further processing of the
current node by the remaining templates in the current template group.

```
Node 1 gets processed by templates A10 and A11
Node 2 gets processed by templates A10 and A11
Node 3 gets processed by templates A10 and A11

Node 1 gets processed by template M10
Node 2 gets processed by template M10
Node 3 gets processed by template M10

Node 1 gets processed by templates Z10 and Z11
Node 2 gets processed by template Z10
Node 3 gets processed by templates Z10 and Z11
```
