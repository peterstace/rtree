# RTree

This package implements an [R-Tree](https://en.wikipedia.org/wiki/R-tree) data
structure.

It follows the approach outlined in [R-Trees - A Dynamic Index Structure For
Spatial Searching](http://www-db.deis.unibo.it/courses/SI-LS/papers/Gut84.pdf).

The implementation is in-memory only, and is designed in such a way that the
internal representation of the R-Tree is exposed. In particular, this allows
the R-Tree to be serialised for storage or transmission. Note that this package
doesn't provide any serialisation - that is an exercise left up to the package
user.
