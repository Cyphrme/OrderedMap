# OrderedMap

`orderedmap` is useful when JSON field ordering is relevant. `encoding/json` has
no way to preserve the order of map keys. See
https://github.com/golang/go/issues/27179.

Using JSONv2 is the future goal, which solves field order, other issues, and has
other best practices.When Go Coze is migrated to JSONv2, as long as JSONv2
provides ordering, orderedmap will be deprecated. See
https://github.com/Cyphrme/Coze/issues/15

### Contributors
Thank you to peterbourgon for contributing!



----------------------------------------------------------------------
# Attribution, Trademark Notice, and License
OrderedMap is released under The 3-Clause BSD License.

"Cyphr.me" is a trademark of Cypherpunk, LLC. The Cyphr.me logo is all rights
reserved Cypherpunk, LLC and may not be used without permission.