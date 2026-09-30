module SelectionDiff exposing (Diff, Item, Line(..), SourceChange, compare, lines)

import Array
import Dict exposing (Dict)
import Packing exposing (Group)
import Set exposing (Set)


type alias Item =
    { name : String, sources : List String }


type alias SourceChange =
    { name : String, before : List String, after : List String }


type alias Diff =
    { added : List Item
    , removed : List Item
    , retained : Int
    , sourceChanges : List SourceChange
    }


sources : List Group -> Dict String (Set String)
sources groups =
    List.foldl
        (\g index ->
            List.foldl
                (\item -> Dict.update item (Maybe.withDefault Set.empty >> Set.insert g.name >> Just))
                index
                g.contents.items
        )
        Dict.empty
        groups


compare : List Group -> List Group -> Diff
compare before after =
    let
        old =
            sources before

        new =
            sources after

        names index =
            index |> Dict.keys |> Set.fromList

        origins index name =
            Dict.get name index |> Maybe.withDefault Set.empty |> Set.toList

        items index keys =
            keys |> Set.toList |> List.map (\name -> Item name (origins index name))

        retained =
            Set.intersect (names old) (names new)
    in
    { added = items new (Set.diff (names new) (names old))
    , removed = items old (Set.diff (names old) (names new))
    , retained = Set.size retained
    , sourceChanges =
        retained
            |> Set.toList
            |> List.filter (\name -> Dict.get name old /= Dict.get name new)
            |> List.map (\name -> SourceChange name (origins old name) (origins new name))
    }


type Line
    = Unchanged String
    | Added String
    | Removed String


{-| A longest-common-subsequence line diff. Work on original text, so comments,
blank lines and requirements are reviewable without reimplementing the parser.
-}
lines : String -> String -> List Line
lines before after =
    let
        split text =
            if text == "" then
                Array.empty

            else
                String.split "\n" text |> Array.fromList

        old =
            split before

        new =
            split after

        oldLength =
            Array.length old

        newLength =
            Array.length new

        get array i =
            Array.get i array |> Maybe.withDefault ""

        lengthAt i j table =
            Dict.get ( i, j ) table |> Maybe.withDefault 0

        fill i j table =
            Dict.insert ( i, j )
                (if get old i == get new j then
                    1 + lengthAt (i + 1) (j + 1) table

                 else
                    max (lengthAt (i + 1) j table) (lengthAt i (j + 1) table)
                )
                table

        lengths =
            List.range 0 (oldLength - 1)
                |> List.reverse
                |> List.foldl
                    (\i table ->
                        List.range 0 (newLength - 1)
                            |> List.reverse
                            |> List.foldl (fill i) table
                    )
                    Dict.empty

        walk i j acc =
            if i >= oldLength && j >= newLength then
                List.reverse acc

            else if i < oldLength && j < newLength && get old i == get new j then
                walk (i + 1) (j + 1) (Unchanged (get old i) :: acc)

            else if i < oldLength && (j >= newLength || lengthAt (i + 1) j lengths >= lengthAt i (j + 1) lengths) then
                walk (i + 1) j (Removed (get old i) :: acc)

            else
                walk i (j + 1) (Added (get new j) :: acc)
    in
    walk 0 0 []
