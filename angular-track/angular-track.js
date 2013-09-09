angular.module('track', [])

.constant('TrackCurrentView', {Seq: 0})

.config(['$httpProvider', 'TrackCurrentView', function($httpProvider, TrackCurrentView) {
  $httpProvider.defaults.transformRequest.push(function(data, headers) {
    headers()['X-Track-View'] = '' + TrackCurrentView.Instance + ' ' + TrackCurrentView.Seq;
    return data;
  });
}])

.run(['$rootScope', 'Track', 'TrackClientData', 'TrackCurrentView', function($rootScope, Track, TrackClientData, TrackCurrentView) {
  $rootScope.$on('$stateChangeStart', function(ev, to, toParams) {
    Track.view(to.name, toParams);
  });
  TrackCurrentView.Instance = TrackClientData.Instance;
}])

.factory('TrackClientConfig', ['$log', '$window', function($log, $window) {
  var data = $window.__trackClientConfig;
  if (!data) {
    $log.error('Missing TrackClientConfig (window.__trackClientConfig is not set)');
  }
  return data;
}])

.factory('TrackClientData', ['$log', '$window', function($log, $window) {
  var data = $window.__trackClientData;
  if (!data) {
    $log.error('Missing TrackClientData (window.__trackClientData is not set)');
  }
  return data;
}])

.factory('Track', ['$http', '$location', '$state', 'TrackClientConfig', 'TrackCurrentView', function($http, $location, $state, cfg, currentView) {
  return {
    view: function(stateName, stateParams) {
      currentView.Seq++;
      // TODO(sqs): this only captures query parameters that are parameters of
      // the state.
      currentView.RequestURI = $state.href(stateName, stateParams);
      currentView.State = stateName;
      currentView.StateParams = stateParams;
      $http.post(cfg.NewViewURL.replace(':instance', currentView.Instance), currentView);
    },
  };
}])

;
