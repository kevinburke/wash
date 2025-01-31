package types

import (
	apitypes "github.com/puppetlabs/wash/api/types"
)

// EntrySchema is a wrapper to apitypes.EntrySchema
type EntrySchema = apitypes.EntrySchema

// Prune prunes s to contain only nodes that satisfy p. It modifies s'
// state so it is not an idempotent operation.
func Prune(s *EntrySchema, p EntrySchemaPredicate, opts Options) *EntrySchema {
	keep := evaluateNodesToKeep(s, p, opts)
	if !keep[s.TypeID()] {
		return nil
	}

	visited := make(map[string]bool)
	var prune func(s *EntrySchema)
	prune = func(s *EntrySchema) {
		if visited[s.TypeID()] {
			return
		}
		visited[s.TypeID()] = true
		var childrenToKeep []*EntrySchema
		for _, child := range s.Children() {
			if keep[child.TypeID()] {
				prune(child)
				childrenToKeep = append(childrenToKeep, child)
			}
		}
		s.SetChildren(childrenToKeep)
	}
	prune(s)

	return s
}

/*
"result" represents the returned value, which is a map of <node> => <keep?>.
Thus, result[N] == false means we prune N, result[N] == true means we keep
it. result[N] == true if:
	* p(N) returns true
	* p(N) returns true for at least one of N's children (ignoring self-loops)
Since entry schemas can contain cycles (e.g. the volume.dir class in the
volume package), and since they do not strictly adhere to a tree structure
(e.g. paths A-B-C and A-D-C are possible), we cannot completely calculate
"result" in a single iteration. For example in the A-B-C, A-D-C case, assume
that p returns false for A, B, C and D. Then we keep B and D if we keep C,
and we keep C if we keep any of its children. If C does not contain a cycle,
then we can recurse back the information to B and D. If C does contain a cycle,
say back to B, then it is difficult, if not impossible, to coordinate all the
updates in a single iteration. However if we let "result[N] == p(N) for all N
such that either p(N) == true or N has no children" be our starting state, then
we can iteratively evaluate result[V] for all other nodes V using the information
provided by a previous iteration. The terminating condition is when an iteration
does not update "result". This is possible if "result" contains all nodes N, or if
there are some indeterminate nodes V. In the latter case, every node M in V is either
part of a cycle, or there is more than one way to get to M from the root. In either
case, we can say that result[M] == false for those nodes.

NOTE: To see why result[M] == false for all M in V, we do a proof by contradiction.
Assume result[M] == true for some M in V, and that M is part of cycle A-B-...-M...-C-A.
Here, we see that one iteration would update all ancestors of M. A subsequent iteration
would update M's descendants starting from C. Thus, all the nodes in the cycle are
determinant, which contradicts our previous assumption of their indeterminacy (since
those nodes are also part of V). The proof for the other case is similar.

NOTE: The starting state formalizes the first condition for result[N] (and also notes
that if N is a leaf, then result[N] == p(N)). Subsequent iterations check the second
condition.
*/
func evaluateNodesToKeep(s *EntrySchema, p EntrySchemaPredicate, opts Options) map[string]bool {
	result := make(map[string]bool)
	visited := make(map[string]bool)

	// Set-up our starting state by evaluating p(N). Note that after this code,
	// result[N] == p(N) for all nodes N such that p(N) == true or N has no
	// children.
	var applyPredicate func(s *EntrySchema)
	applyPredicate = func(s *EntrySchema) {
		if _, ok := visited[s.TypeID()]; ok {
			return
		}
		// Set the metadata schema prior to evaluating the predicate
		metadataSchema := s.MetaAttributeSchema()
		if opts.Fullmeta && s.MetadataSchema() != nil {
			metadataSchema = s.MetadataSchema()
		}
		s.SetMetadataSchema(metadataSchema)
		result[s.TypeID()] = p.P(s)
		if !result[s.TypeID()] && len(s.Children()) > 0 {
			delete(result, s.TypeID())
		}
		visited[s.TypeID()] = true
		for _, child := range s.Children() {
			applyPredicate(child)
		}
	}
	applyPredicate(s)

	// Now we iteratively update "result".
	var updateResult func(s *EntrySchema)
	updateResult = func(s *EntrySchema) {
		if _, ok := visited[s.TypeID()]; ok {
			return
		}
		visited[s.TypeID()] = true
		for _, child := range s.Children() {
			updateResult(child)
			if result[child.TypeID()] {
				result[s.TypeID()] = true
			}
		}
	}
	for {
		visited = make(map[string]bool)
		prevNumNodes := len(result)
		updateResult(s)
		newNodes := len(result) - prevNumNodes
		if newNodes <= 0 {
			// We've reached the terminating condition. Note that result[V] == false
			// for an indeterminate node V since false is the 0-value for a "bool" type.
			// Thus, we can just return here.
			return result
		}
	}
}
