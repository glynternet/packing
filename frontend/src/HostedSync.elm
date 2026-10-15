module HostedSync exposing (..)

import Dict exposing (Dict)
import Json.Decode as D
import Json.Encode as E
import Packing


type alias Item =
    { id : String, name : String, packed : Bool, revision : Int }


type alias Trip =
    { id : String
    , epoch : String
    , revision : Int
    , draft : String
    , draftRevision : Int
    , generation : Int
    , snapshot : Packing.Snapshot
    , items : Dict String Item
    }


type alias Write =
    { mutationId : String
    , kind : String
    , etag : String
    , epoch : String
    , generation : Int
    , draftRevision : Int
    , text : String
    , item : String
    , packed : Bool
    }


type alias Ack =
    { draftRevision : Int, generation : Int, revision : Int, trip : Trip }


tag : Trip -> String -> Int -> String
tag trip resource revision =
    "\"" ++ trip.epoch ++ ":" ++ resource ++ ":" ++ String.fromInt revision ++ "\""


write : String -> String -> Trip -> Write
write id kind trip =
    { mutationId = id
    , kind = kind
    , etag =
        tag trip
            (if kind == "draft" then
                "draft"

             else
                "snapshot"
            )
            (if kind == "draft" then
                trip.draftRevision

             else
                trip.generation
            )
    , epoch = trip.epoch
    , generation = trip.generation
    , draftRevision = trip.draftRevision
    , text = trip.draft
    , item = ""
    , packed = False
    }


encodeWrite : Write -> E.Value
encodeWrite w =
    E.object
        [ ( "mutationId", E.string w.mutationId )
        , ( "kind", E.string w.kind )
        , ( "etag", E.string w.etag )
        , ( "epoch", E.string w.epoch )
        , ( "generation", E.int w.generation )
        , ( "draftRevision", E.int w.draftRevision )
        , ( "text", E.string w.text )
        , ( "item", E.string w.item )
        , ( "packed", E.bool w.packed )
        ]


writeDecoder : D.Decoder Write
writeDecoder =
    D.map2 (\partial packed -> partial packed)
        (D.map8 Write
            (D.field "mutationId" D.string)
            (D.field "kind" D.string)
            (D.field "etag" D.string)
            (D.field "epoch" D.string)
            (D.field "generation" D.int)
            (D.field "draftRevision" D.int)
            (D.field "text" D.string)
            (D.field "item" D.string)
        )
        (D.field "packed" D.bool)


tripDecoder : D.Decoder Trip
tripDecoder =
    D.map8 Trip
        (D.field "id" D.string)
        (D.field "epoch" D.string)
        (D.field "revision" D.int)
        (D.field "draft" D.string)
        (D.field "draftRevision" D.int)
        (D.field "generation" D.int)
        (D.field "snapshot" Packing.decodeSnapshot)
        (D.field "items"
            (D.dict
                (D.map4 Item
                    (D.field "id" D.string)
                    (D.field "name" D.string)
                    (D.field "packed" D.bool)
                    (D.field "revision" D.int)
                )
            )
        )


ackDecoder : D.Decoder Ack
ackDecoder =
    D.map4 Ack
        (D.at [ "outcome", "draftRevision" ] D.int)
        (D.at [ "outcome", "generation" ] D.int)
        (D.at [ "outcome", "revision" ] D.int)
        (D.field "trip" tripDecoder)


{-| Never regress a confirmed aggregate when a delayed receipt/fetch arrives.
Epoch changes are handled separately as explicit restore recovery.
-}
newest : Trip -> Maybe Trip -> Trip
newest incoming previous =
    case previous of
        Just old ->
            if old.epoch == incoming.epoch && old.revision > incoming.revision then
                old

            else
                incoming

        Nothing ->
            incoming
