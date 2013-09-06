angular.module('track', [])

.constant('TrackCurrentView', {Seq: -1})

.config(['$httpProvider', 'TrackCurrentView', function($httpProvider, TrackCurrentView) {
  $httpProvider.defaults.transformRequest.push(function(data, headers) {
    headers()['X-Track-View'] = '' + TrackCurrentView.Win + ' ' + TrackCurrentView.Seq;
    return data;
  });
}])

.run(['$rootScope', 'Track', 'TrackClientData', 'TrackCurrentView', function($rootScope, Track, TrackClientData, TrackCurrentView) {
  $rootScope.$on('$stateChangeStart', function(ev, to, toParams) {
    Track.view(to.name, toParams);
  });
  TrackCurrentView.Win = TrackClientData.Win;
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

.factory('Track', ['$http', 'TrackClientConfig', 'TrackCurrentView', function($http, cfg, currentView) {
  return {
    view: function(stateName, stateParams) {
      currentView.Seq++;
      currentView.State = stateName;
      currentView.Params = stateParams;
      $http.post(cfg.NewViewURL.replace(':win', currentView.Win), currentView);
    },
  };
}])

;
