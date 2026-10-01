module PackingTest exposing (tests)

import Expect
import Packing exposing (Group)
import Test exposing (Test, describe, test)


group : String -> List String -> List String -> Group
group name refs items =
    { name = name, contents = { refs = refs, items = items } }


tests : Test
tests =
    describe "Resolved group items"
        [ test "includes own and nested items in name order, excluding ancestors and unrelated groups" <|
            \_ ->
                let
                    groups =
                        [ group "parent" [ "a" ] [ "parent item" ]
                        , group "a" [ "b" ] [ "a_2", "a_1" ]
                        , group "b" [] [ "b_2", "b_1" ]
                        , group "other" [] [ "unrelated item" ]
                        ]
                in
                Expect.equal
                    ( Just [ "a_1", "a_2", "b_1", "b_2" ], Just [ "b_1", "b_2" ] )
                    ( Packing.resolvedItems "a" groups, Packing.resolvedItems "b" groups )
        , test "deduplicates repeated items and shared descendants at every depth" <|
            \_ ->
                [ group "root" [ "left", "right", "left" ] [ "shared", "shared" ]
                , group "left" [ "middle" ] [ "left item" ]
                , group "middle" [ "leaf" ] []
                , group "right" [ "leaf" ] [ "shared" ]
                , group "leaf" [] [ "leaf item", "shared" ]
                ]
                    |> Packing.resolvedItems "root"
                    |> Expect.equal (Just [ "leaf item", "left item", "shared" ])
        , test "cycles and self references terminate without losing reachable items" <|
            \_ ->
                [ group "a" [ "a", "b" ] [ "a_1" ]
                , group "b" [ "c" ] [ "b_1" ]
                , group "c" [ "a" ] [ "c_1" ]
                ]
                    |> Packing.resolvedItems "a"
                    |> Expect.equal (Just [ "a_1", "b_1", "c_1" ])
        , test "container and singleton chains resolve even when they have no direct items" <|
            \_ ->
                [ group "container" [ "passthrough" ] []
                , group "passthrough" [ "singleton" ] []
                , group "singleton" [] [ "item" ]
                ]
                    |> Packing.resolvedItems "container"
                    |> Expect.equal (Just [ "item" ])
        , test "distinguishes empty groups from missing groups" <|
            \_ ->
                let
                    groups =
                        [ group "empty" [] [] ]
                in
                Expect.equal ( Just [], Nothing )
                    ( Packing.resolvedItems "empty" groups, Packing.resolvedItems "missing" groups )
        , test "missing child references do not discard known items" <|
            \_ ->
                [ group "a" [ "missing", "b" ] [ "a_1" ], group "b" [] [ "b_1" ] ]
                    |> Packing.resolvedItems "a"
                    |> Expect.equal (Just [ "a_1", "b_1" ])
        ]
