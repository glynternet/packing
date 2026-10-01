module Packing exposing (ContentsDefinition, Group, Snapshot, canonical, decodeGroups, decodeSnapshot, encodeSnapshot, nullableList, resolvedItems)

import Dict
import Json.Decode as Decode
import Json.Encode as Encode
import Set


type alias Group =
    { name : String, contents : ContentsDefinition }


type alias ContentsDefinition =
    { refs : List String, items : List String }


type alias Snapshot =
    { selection : String, groups : List Group }


{-| Collect unique items from a group and every descendant in the raw snapshot.
Following only child references excludes ancestors and unrelated groups. A visited
set handles shared descendants and cycles; Set.toList gives stable name ordering.
Nothing distinguishes a missing root from a group that resolves to no items.
-}
resolvedItems : String -> List Group -> Maybe (List String)
resolvedItems name groups =
    let
        index =
            groups |> List.map (\group -> ( group.name, group.contents )) |> Dict.fromList

        collect pending visited items =
            case pending of
                [] ->
                    Set.toList items

                current :: rest ->
                    if Set.member current visited then
                        collect rest visited items

                    else
                        case Dict.get current index of
                            Nothing ->
                                collect rest (Set.insert current visited) items

                            Just contents ->
                                collect (contents.refs ++ rest)
                                    (Set.insert current visited)
                                    (List.foldl Set.insert items contents.items)
    in
    Dict.get name index
        |> Maybe.map (\_ -> collect [ name ] Set.empty Set.empty)


nullableList : Decode.Decoder a -> Decode.Decoder (List a)
nullableList decoder =
    Decode.nullable (Decode.list decoder) |> Decode.map (Maybe.withDefault [])


decodeGroups : Decode.Decoder (List Group)
decodeGroups =
    Decode.list
        (Decode.map2 Group
            (Decode.field "name" Decode.string)
            (Decode.field "contents"
                (Decode.map2 ContentsDefinition
                    (Decode.field "refs" (nullableList Decode.string))
                    (Decode.field "items" (nullableList Decode.string))
                )
            )
        )


decodeSnapshot : Decode.Decoder Snapshot
decodeSnapshot =
    Decode.map2 Snapshot
        (Decode.field "selection" Decode.string)
        (Decode.field "groups" decodeGroups)


encodeSnapshot : Snapshot -> Encode.Value
encodeSnapshot snapshot =
    let
        strings =
            Encode.list Encode.string

        group g =
            Encode.object
                [ ( "name", Encode.string g.name )
                , ( "contents"
                  , Encode.object
                        [ ( "refs", strings g.contents.refs )
                        , ( "items", strings g.contents.items )
                        ]
                  )
                ]
    in
    Encode.object
        [ ( "selection", Encode.string snapshot.selection )
        , ( "groups", Encode.list group snapshot.groups )
        ]


{-| Ignore server map ordering and repeated memberships when comparing results.
Keep raw groups in snapshots so display transforms never alter accepted data.
-}
canonical : List Group -> List ( String, List String, List String )
canonical groups =
    let
        sortedSet =
            Set.fromList >> Set.toList
    in
    groups
        |> List.map (\g -> ( g.name, sortedSet g.contents.refs, sortedSet g.contents.items ))
        |> List.sort
