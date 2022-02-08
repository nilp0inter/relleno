port module Main exposing (..)

import Bootstrap.Button as Button
import Bootstrap.CDN as CDN
import Bootstrap.Grid as Grid
import Bootstrap.Navbar as Navbar
import Bootstrap.Utilities.Spacing as Spacing
import Browser
import Debug
import Html exposing (Html, button, div, footer, h1, text)
import Html.Attributes exposing (class, href, id, style)
import Html.Events exposing (onClick)
import Http
import Json.Decode as Decode
import List
import Platform.Cmd exposing (batch)
import Task exposing (Task)



-- MAIN


main =
    Browser.document
        { init = init
        , subscriptions = subscriptions
        , update = update
        , view = view
        }



-- PORTS


port createEditor : ServerData -> Cmd msg


port editorInitializer : (() -> msg) -> Sub msg


port editorChanger : (Bool -> msg) -> Sub msg


port getEditorValue : () -> Cmd msg


port editorValueGetter : (Maybe Decode.Value -> msg) -> Sub msg



-- MODEL


type alias Action =
    { name : String
    , text : String
    , color : String
    }


type alias Config =
    { editor : Maybe Decode.Value
    , actions : List Action
    }


type alias ServerData =
    { config : Config
    , schema : Decode.Value
    , document : Decode.Value
    }


type FormState
    = Loading
    | InitializingEditor ServerData
    | LoadError
    | Editing
    | SendingAction
    | Finishing


type alias Model =
    { navbarState : Navbar.State
    , formState : FormState
    , actions : List Action
    , formValid : Bool
    , requestedAction : String
    }


loadServerData =
    Task.attempt GotServerData (Task.map3 ServerData getConfig getSchema getDocument)


init : String -> ( Model, Cmd Msg )
init _ =
    let
        ( navbarState, navbarCmd ) =
            Navbar.initialState NavbarMsg
    in
    ( { navbarState = navbarState
      , formState = Loading
      , actions = []
      , formValid = False
      , requestedAction = ""
      }
    , Cmd.batch [ loadServerData, navbarCmd ]
    )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Navbar.subscriptions model.navbarState NavbarMsg
        , editorInitializer (\_ -> EditorInitialized)
        , editorChanger EditorChanged
        , editorValueGetter GotEditorValue
        ]



-- UPDATE


type Msg
    = GotServerData (Result Http.Error ServerData)
    | NavbarMsg Navbar.State
    | EditorInitialized
    | EditorChanged Bool
    | GetEditorValue String
    | GotEditorValue (Maybe Decode.Value)
    | ActionSent (Result Http.Error ())


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        GotServerData x ->
            case x of
                Ok cfg ->
                    ( { model | formState = InitializingEditor cfg }, createEditor cfg )

                Err err ->
                    ( { model | formState = LoadError }, Cmd.none )

        NavbarMsg state ->
            ( { model | navbarState = state }, Cmd.none )

        EditorInitialized ->
            case model.formState of
                InitializingEditor cfg ->
                    ( { model | formState = Editing, actions = cfg.config.actions }, Cmd.none )

                _ ->
                    ( { model | formState = LoadError }, Cmd.none )

        EditorChanged isValid ->
            ( { model | formValid = isValid }, Cmd.none )

        GetEditorValue action ->
            ( { model | formValid = False, requestedAction = action }, getEditorValue () )

        GotEditorValue value ->
            case value of
                Just data ->
                    ( { model | formValid = True }, sendAction model.requestedAction data )

                Nothing ->
                    ( { model | formValid = False }, Cmd.none )

        ActionSent response ->
            case response of
                Ok _ ->
                    ( { model | formState = Finishing }, Cmd.none )

                Err _ ->
                    ( { model | formState = LoadError }, Cmd.none )



-- VIEW
-- view : Model -> List (Html msg)


toButtonColor colorName =
    case colorName of
        "primary" ->
            Button.primary

        "secondary" ->
            Button.secondary

        "success" ->
            Button.success

        "info" ->
            Button.info

        "warning" ->
            Button.warning

        "danger" ->
            Button.danger

        "light" ->
            Button.light

        _ ->
            Button.dark


toMenuButton isDisabled txt name color =
    Button.button
        [ toButtonColor color
        , Button.attrs [ Spacing.ml2 ]
        , Button.disabled <| not isDisabled
        , Button.onClick <| GetEditorValue name
        ]
        [ text txt ]


menu : Model -> Html Msg
menu model =
    let
        buttons =
            List.map (\a -> toMenuButton model.formValid a.text a.name a.color) model.actions
    in
    Navbar.config NavbarMsg
        |> Navbar.withAnimation
        |> Navbar.fixTop
        |> Navbar.dark
        |> Navbar.brand [ href "#" ] [ text "Relleno" ]
        |> Navbar.customItems
            [ Navbar.formItem []
                (if model.formState == Finishing then
                    []

                 else
                    buttons
                )
            ]
        |> Navbar.view model.navbarState


view : Model -> Browser.Document Msg
view model =
    let
        errors =
            if model.formState == LoadError then
                [ div [] [ text "error" ] ]

            else
                []
    in
    { title = "Relleno"
    , body =
        [ menu model
        , Grid.container [ style "margin-top" "80px" ]
            [ CDN.stylesheet -- creates an inline style node with the Bootstrap CSS
            , Grid.row []
                [ Grid.col []
                    (case model.formState of
                        Finishing ->
                            [ text "Finish!" ]

                        _ ->
                            errors ++ [ div [ id "editor" ] [] ]
                    )
                ]
            ]
        ]
    }



-- MESSAGES


handleJsonResponse : Decode.Decoder a -> Http.Response String -> Result Http.Error a
handleJsonResponse decoder response =
    case response of
        Http.BadUrl_ url ->
            Err (Http.BadUrl url)

        Http.Timeout_ ->
            Err Http.Timeout

        Http.BadStatus_ { statusCode } _ ->
            Err (Http.BadStatus statusCode)

        Http.NetworkError_ ->
            Err Http.NetworkError

        Http.GoodStatus_ _ body ->
            case Decode.decodeString decoder body of
                Err _ ->
                    Err (Http.BadBody body)

                Ok result ->
                    Ok result


decodeAction =
    Decode.map3 Action
        (Decode.field "name" Decode.string)
        (Decode.field "text" Decode.string)
        (Decode.field "color" Decode.string)


decodeConfig =
    Decode.map2 Config
        (Decode.maybe (Decode.field "editor" Decode.value))
        (Decode.field "actions" (Decode.list decodeAction))


getConfig : Task Http.Error Config
getConfig =
    Http.task
        { method = "GET"
        , headers = []
        , url = "/api/doc/f653e4b3-f4c8-4384-9f0a-2fa4e9375804/config"
        , body = Http.emptyBody
        , resolver = Http.stringResolver <| handleJsonResponse <| decodeConfig
        , timeout = Nothing
        }


getDocument : Task Http.Error Decode.Value
getDocument =
    Http.task
        { method = "GET"
        , headers = []
        , url = "/api/doc/f653e4b3-f4c8-4384-9f0a-2fa4e9375804/document"
        , body = Http.emptyBody
        , resolver = Http.stringResolver <| handleJsonResponse <| Decode.value
        , timeout = Nothing
        }


getSchema : Task Http.Error Decode.Value
getSchema =
    Http.task
        { method = "GET"
        , headers = []
        , url = "/api/doc/f653e4b3-f4c8-4384-9f0a-2fa4e9375804/schema"
        , body = Http.emptyBody
        , resolver = Http.stringResolver <| handleJsonResponse <| Decode.value
        , timeout = Nothing
        }


sendAction : String -> Decode.Value -> Cmd Msg
sendAction action data =
    Http.post
        { url = "/api/doc/f653e4b3-f4c8-4384-9f0a-2fa4e9375804/action/" ++ action
        , body = Http.jsonBody data
        , expect = Http.expectWhatever ActionSent
        }
