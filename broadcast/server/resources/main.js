var app = angular.module('test', []);

app.directive('enterSubmit', function() {
    return function(scope, element, attrs) {
        element.bind("keydown keypress", function(event) {
            if(event.which === 13) {
                scope.$apply(function(){
                    scope.$eval(attrs.enterSubmit, {'event': event});
                });
                event.preventDefault();
            }
        });
    };
});

app.directive('msg', function() {
    return {
        restrict: 'E',
        scope: {
            msg: "=message"
        },
        template: '<p class="msg"><span class="{{msg.sender.toLowerCase()}}">{{msg.sender}}</span>: {{msg.text}}</p>'
    };
});


app.factory('webSocket', ['$rootScope', function($rootScope) {
    var ws = new WebSocket('ws://localhost:8081/socket');
    var service = {
        messages: [],
        send: function(msg) {
            ws.send(msg);
        }
    }
    ws.onopen = function(){
        console.log('Socket opened');
    };
    ws.onclose = function(){
        console.log('Socket closed');
    };
    ws.onerror = function(evt){
        console.log('Socket error');
        console.log(evt);
    };
    ws.onmessage = function(msg){
        service.messages.push({sender: 'Server', text: msg.data});
        $rootScope.$broadcast( 'messages.update' );
    };
    return service;
}]);

app.controller('TestCtrl', ['$scope', 'webSocket', function ($scope, webSocket) {
    $scope.send = function(msg) {
        $scope.messages.push({sender: 'Me', text: msg});
        webSocket.send(msg + '\n');
    };
    $scope.$on('messages.update', function() {
        $scope.$apply(function() {
            $scope.messages = webSocket.messages;
        });
    });
    $scope.messages = webSocket.messages;
}]);
