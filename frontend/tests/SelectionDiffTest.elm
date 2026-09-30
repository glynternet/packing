module SelectionDiffTest exposing (tests)

import Expect
import Fuzz
import Packing exposing (Group)
import SelectionDiff exposing (Line(..))
import Set
import Test exposing (Test, describe, fuzz2, test)


group : String -> List String -> Group
group name items =
    { name = name, contents = { refs = [], items = items } }


tests : Test
tests =
    describe "Resolved item diff"
        [ test "removing one of two sources retains the shared item" <|
            \_ ->
                SelectionDiff.compare
                    [ group "sun" [ "Sunscreen", "Sunglasses" ], group "hiking" [ "Sunscreen" ] ]
                    [ group "hiking" [ "Sunscreen" ] ]
                    |> Expect.equal
                        { added = []
                        , removed = [ { name = "Sunglasses", sources = [ "sun" ] } ]
                        , retained = 1
                        , sourceChanges = [ { name = "Sunscreen", before = [ "hiking", "sun" ], after = [ "hiking" ] } ]
                        }
        , test "counts unique items and sorts them consistently" <|
            \_ ->
                SelectionDiff.compare [] [ group "b" [ "Towel", "Hat", "Hat" ], group "a" [ "Hat" ] ]
                    |> .added
                    |> Expect.equal
                        [ { name = "Hat", sources = [ "a", "b" ] }
                        , { name = "Towel", sources = [ "b" ] }
                        ]
        , test "removing a loose item still supplied by a group is only a source change" <|
            \_ ->
                let
                    diff =
                        SelectionDiff.compare
                            [ group "Individual Items" [ "Towel" ], group "swimming" [ "Towel" ] ]
                            [ group "swimming" [ "Towel" ] ]
                in
                Expect.equal ( [], [], 1 ) ( diff.added, diff.removed, diff.retained )
        , fuzz2 (Fuzz.list Fuzz.string) (Fuzz.list Fuzz.string) "applying item additions/removals reconstructs the proposed item set" <|
            \before after ->
                let
                    diff =
                        SelectionDiff.compare [ group "before" before ] [ group "after" after ]

                    result =
                        Set.diff (Set.fromList before) (Set.fromList (List.map .name diff.removed))
                            |> Set.union (Set.fromList (List.map .name diff.added))
                in
                Expect.equal (Set.fromList after) result
        , test "line diff retains comments, requirements and unchanged context" <|
            \_ ->
                SelectionDiff.lines "# trip\nref: sun\nreq: passport\n" "# trip\nref: cold\nreq: passport\n"
                    |> Expect.equal
                        [ Unchanged "# trip", Removed "ref: sun", Added "ref: cold", Unchanged "req: passport", Unchanged "" ]
        , test "empty selections have no spurious blank-line diff" <|
            \_ ->
                Expect.equal ( [], [ Removed "Towel" ], [ Added "Towel" ] )
                    ( SelectionDiff.lines "" "", SelectionDiff.lines "Towel" "", SelectionDiff.lines "" "Towel" )
        , fuzz2 (Fuzz.stringOfLengthBetween 0 60) (Fuzz.stringOfLengthBetween 0 60) "line diff reconstructs both original texts" <|
            \before after ->
                let
                    diff =
                        SelectionDiff.lines before after

                    old line =
                        case line of
                            Added _ ->
                                Nothing

                            Removed value ->
                                Just value

                            Unchanged value ->
                                Just value

                    new line =
                        case line of
                            Removed _ ->
                                Nothing

                            Added value ->
                                Just value

                            Unchanged value ->
                                Just value
                in
                Expect.equal ( before, after )
                    ( diff |> List.filterMap old |> String.join "\n", diff |> List.filterMap new |> String.join "\n" )
        ]
