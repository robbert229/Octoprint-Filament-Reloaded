# coding=utf-8
from __future__ import absolute_import

import octoprint.plugin
from octoprint.events import Events
from time import sleep


class FilamentReloadedServerPlugin(octoprint.plugin.StartupPlugin,
								   octoprint.plugin.EventHandlerPlugin,
								   octoprint.plugin.TemplatePlugin,
								   octoprint.plugin.SettingsPlugin):

	def initialize(self):
		self._logger.info("Filament Sensor Server initialized")

	def on_after_startup(self):
		self._logger.info("Filament Sensor Reloaded Server started")
		# self._setup_sensor()

	def get_settings_defaults(self):
		return dict(
			url='http://localhost:3278/api/check/2',
			no_filament_gcode='',
			pause_print=True,
		)

	def on_settings_save(self, data):
		octoprint.plugin.SettingsPlugin.on_settings_save(self, data)

	def get_template_configs(self):
		return [dict(type="settings", custom_bindings=False)]

	def get_update_information(self):
		return dict(
			octoprint_filament=dict(
				displayName="Filament Sensor Reloaded Server",
				displayVersion=self._plugin_version,

				# version check: github repository
				type="github_release",
				user="robbert229",
				repo="Octoprint-Filament-Reloaded-Server",
				current=self._plugin_version,

				# update method: pip
				pip="https://github.com/robbert229/Octoprint-Filament-Reloaded-Server/archive/{target_version}.zip"
			)
		)


__plugin_name__ = "Filament Sensor Reload Server"
__plugin_version__ = "0.0.1"


def __plugin_load__():
	global __plugin_implementation__
	__plugin_implementation__ = FilamentReloadedServerPlugin()

	global __plugin_hooks__
	__plugin_hooks__ = {
		"octoprint.plugin.softwareupdate.check_config": __plugin_implementation__.get_update_information
	}
