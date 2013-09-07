angular.module('example', [
  'ngResource',
  'ui.router.compat',
  'track',
])

.config(function($locationProvider, $routeProvider, $stateProvider) {
  $locationProvider.html5Mode(true);

  $routeProvider
    .when('/', {
      redirectTo: '/contacts',
    });

  $stateProvider

    .state('contacts', {
      url: '/contacts',
      views: {
        '': {
          templateUrl: '/static/contacts.tpl.html',
        },
        'list@contacts': {
          controller: 'ContactsListCtrl',
          templateUrl: '/static/contacts.list.tpl.html',
        },
      },
    })
    .state('contacts.detail', {
      url: '/{id}',
      resolve: {
        contact: function($stateParams, Contact) {
          return Contact.get({id: $stateParams.id}).$promise;
        },
      },
      views: {
        'detail@contacts': {
          controller: 'ContactDetailCtrl',
          templateUrl: '/static/contacts.detail.tpl.html',
        },
      },
    })

  ;
})

.controller('ContactsListCtrl', function($scope, Contact) {
  $scope.contacts = Contact.query();
})

.controller('ContactDetailCtrl', function($scope, Contact, contact) {
  $scope.contact = contact;
  $scope.reload = function() {
    $scope.contact = Contact.get({id: contact.ID});
  };
})

.factory('Contact', function($resource) {
  return $resource('/api/contacts/:id');
})

.controller('ConfigCtrl', function($scope, Instance, TrackClientConfig, TrackCurrentView) {
  $scope.TrackClientConfig = TrackClientConfig;
  $scope.TrackCurrentView = TrackCurrentView;
  $scope.instance = Instance.get({instance: TrackCurrentView.Instance});
})

.controller('CallsCtrl', function($rootScope, $scope, $timeout, Call, TrackCurrentView) {
  function reload() {
    Call.query({instance: TrackCurrentView.Instance, seq: TrackCurrentView.Seq}).$promise.then(function(calls) {
      $scope.calls = calls;
    });
  }
  $rootScope.$on('$stateChangeSuccess', function() {
    reload();
  });
  $timeout(function repeat() {
    reload();
    $timeout(repeat, 1000);
  },1000);
})

.controller('ViewsCtrl', function($rootScope, $scope, View, TrackCurrentView) {
  $rootScope.$on('$stateChangeSuccess', function() {
    $scope.views = View.query({instance: TrackCurrentView.Instance});
  });
})

.factory('Instance', function($resource) {
  return $resource('/api/track/instances/:instance');
})

.factory('Call', function($resource) {
  return $resource('/api/track/instances/:instance/views/:seq/calls/:call');
})

.factory('View', function($resource) {
  return $resource('/api/track/instances/:instance/views/:seq');
})

;
